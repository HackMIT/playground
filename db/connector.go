package db

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/techx/playground/config"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"

	mapset "github.com/deckarep/golang-set"
	"github.com/go-redis/redis/v7"
)

const ingestClientName string = "ingest"

var (
	ingestID string
	instance *redis.Client
	leader   bool
	psc      *redis.PubSub
)

// Init creates the database connection
func Init(shouldReset bool) {
	config := config.GetConfig()

	dbAddr := os.Getenv("DATABASE_ADDR")
	dbPass := os.Getenv("DATABASE_PASS")

	if dbAddr == "" {
		instance = redis.NewClient(&redis.Options{
			Addr:     config.GetString("db.addr"),
			Password: config.GetString("db.password"),
			DB:       config.GetInt("db.db"),
		})
	} else {
		instance = redis.NewClient(&redis.Options{
			Addr:     dbAddr,
			Password: dbPass,
			DB:       0,
		})
	}

	if shouldReset {
		reset()
	}

	// Save our ingest ID
	ingestID = uuid.New().String()

	// Initialize jukebox
	// TODO: Make sure this works correctly when there are multiple ingests
	instance.HSet("queuestatus", utils.StructToMap(models.QueueStatus{SongEnd: time.Now()}))

	// Update TIM the beaver
	character := *models.NewTIMCharacter()
	instance.HSet("character:tim", utils.StructToMap(character))
	instance.SAdd("room:home:characters", "tim")
}

func GetIngestID() string {
	return ingestID
}

func GetInstance() *redis.Client {
	return instance
}

func ListenForUpdates(callback func(msg []byte)) {
	ingests, err := instance.LRange("ingests", 0, -1).Result()

	if err != nil {
		panic(err)
	}

	// Let other ingest servers know about this one
	instance.Publish("ingest", ingestID)

	// subscribe to existing ingests, send id to master
	psc = instance.Subscribe(append(ingests, "ingest", "all")...)

	// Add this ingest to the ingests set
	instance.RPush("ingests", ingestID)

	for {
		msg, err := psc.ReceiveMessage()

		if err != nil {
			// Stop server if we disconnect from Redis
			panic(err)
		}

		if msg.Channel == "ingest" {
			// If this is a new ingest server, subscribe to it
			psc.Subscribe(msg.Payload)
		} else {
			callback([]byte(msg.Payload))
		}
	}
}

func MonitorLeader() {
	i := 0

	for range time.NewTicker(time.Second).C {
		// Set our name so we can identify this client as an ingest
		// Not sure why, but the client names occasionally get reset -- let's do it every second
		pip := instance.Pipeline()
		pip.Process(redis.NewStringCmd("client", "setname", ingestID))
		clientsCmd := instance.ClientList()
		pip.Exec()

		clientsRes, _ := clientsCmd.Result()
		clients := strings.Split(clientsRes, "\n")

		connectedIngestIDs := mapset.NewSet()

		for _, client := range clients {
			clientParts := strings.Split(client, " ")

			if len(clientParts) < 4 {
				// Probably a newline, invalid client
				continue
			}

			clientName := strings.Split(clientParts[3], "=")[1]

			if len(clientName) != 36 {
				// This isn't a uuid, not an ingest
				continue
			}

			connectedIngestIDs.Add(clientName)
		}

		var leaderID string
		ingestIDs, _ := instance.LRange("ingests", 0, -1).Result()

		for _, ingestID := range ingestIDs {
			if connectedIngestIDs.Contains(ingestID) {
				leaderID = ingestID
				break
			}
		}

		// If we're not the leader, don't do any leader actions
		if leaderID != ingestID {
			fmt.Println("not leader")
			continue
		}

		//////////////////////////////////////////////
		// ALL CODE BELOW IS ONLY RUN ON THE LEADER //
		//////////////////////////////////////////////

		// Mark ourselves as the leader
		leader = true

		// Take care of ingest servers that got disconnected
		for _, ingestID := range ingestIDs {
			if connectedIngestIDs.Contains(ingestID) {
				// This ingest is currently connected to Redis
				continue
			}

			fmt.Println("removing", ingestID)

			// Remove characters connected to this ingest from their rooms
			characters, _ := instance.SMembers("ingest:" + ingestID + ":characters").Result()

			pip := instance.Pipeline()
			roomCmds := make([]*redis.StringCmd, len(characters))

			for i, characterID := range characters {
				roomCmds[i] = pip.HGet("character:"+characterID, "room")
			}

			pip.Exec()
			pip = instance.Pipeline()

			for i, roomCmd := range roomCmds {
				room, _ := roomCmd.Result()
				pip.SRem("room:"+room, characters[i])
			}

			pip.Exec()

			// Ingest has been taken care of, remove from set
			instance.LRem("ingests", 0, ingestID)
		}

		// Get song queue status
		queueRes, _ := instance.HGetAll("queuestatus").Result()

		var queueStatus models.QueueStatus
		utils.Bind(queueRes, &queueStatus)

		songEnd := queueStatus.SongEnd

		// If current song ended, start next song (if there is one)
		if songEnd.Before(time.Now()) {
			queueLength, _ := instance.SCard("songs").Result()

			if queueLength > 0 {
				// Pop the next song off the queue
				fmt.Println("queue length > 0")
				songID, _ := instance.LPop("songs").Result()

				songRes, _ := instance.HGetAll("song:" + songID).Result()
				var song models.Song
				utils.Bind(songRes, &song)

				// Update queue status to reflect new song
				endTime := time.Now().Add(time.Second * time.Duration(song.Duration))
				newStatus := models.QueueStatus{endTime}
				instance.HSet("queuestatus", utils.StructToMap(newStatus))

				// Send song packet to ingests
				songPacket := map[string]interface{}{
					"type": "song",
					"song": song,
					"playing": true,
					"endTime": endTime,
				}

				fmt.Println("about to send")
				data, _ := json.Marshal(songPacket)
				pip.Publish("all", data)
				pip.Exec()
			}
		}

		if i%15 == 0 {
			characterRes, _ := instance.HGetAll("character:tim").Result()
			var tim models.Character
			tim.ID = "tim"
			utils.Bind(characterRes, &tim)

			whatToDo := rand.Float64()

			if whatToDo < 0.33 {
				hallwaysRes, _ := instance.SMembers("room:" + tim.Room + ":hallways").Result()

				if len(hallwaysRes) == 0 {
					continue
				}

				hallwayID := hallwaysRes[rand.Intn(len(hallwaysRes))]

				hallwayRes, _ := instance.HGetAll("hallway:" + hallwayID).Result()
				var hallway models.Hallway
				utils.Bind(hallwayRes, &hallway)

				movePacket := map[string]interface{}{
					"type": "move",
					"id":   "tim",
					"room": tim.Room,
					"x":    hallway.X,
					"y":    hallway.Y,
				}
				data, _ := json.Marshal(movePacket)

				pip := instance.Pipeline()
				pip.HSet("character:tim", "x", hallway.X)
				pip.HSet("character:tim", "y", hallway.Y)
				pip.Publish("all", data)
				pip.Exec()

				time.AfterFunc(4*time.Second, func() {
					pip := instance.Pipeline()
					pip.SRem("room:"+tim.Room+":characters", "tim")
					pip.SAdd("room:"+hallway.To+":characters", "tim")
					pip.HSet("character:tim", "room", hallway.To)
					pip.HSet("character:tim", "x", hallway.ToX)
					pip.HSet("character:tim", "y", hallway.ToY)

					timData, _ := tim.MarshalBinary()
					var newTimData map[string]interface{}
					json.Unmarshal(timData, &newTimData)

					teleportPacket := map[string]interface{}{
						"type":      "teleport",
						"character": newTimData,
						"from":      tim.Room,
						"to":        hallway.To,
						"x":         hallway.ToX,
						"y":         hallway.ToY,
					}

					data, _ = json.Marshal(teleportPacket)
					pip.Publish("all", data)
					pip.Exec()
				})
			} else {
				x := rand.Float64()
				y := rand.Float64()

				movePacket := map[string]interface{}{
					"type": "move",
					"id":   "tim",
					"room": tim.Room,
					"x":    x,
					"y":    y,
				}
				data, _ := json.Marshal(movePacket)

				pip := instance.Pipeline()
				pip.HSet("character:tim", "x", x)
				pip.HSet("character:tim", "y", y)
				pip.Publish("all", data)
				pip.Exec()
			}
		}

		i++
	}
}

func Publish(msg interface{}) {
	instance.Publish(ingestID, msg)
}
