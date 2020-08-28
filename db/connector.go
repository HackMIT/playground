package db

import (
	"encoding"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"

	mapset "github.com/deckarep/golang-set"
	"github.com/go-redis/redis/v7"
)

const ingestClientName string = "ingest"

var (
	ingestID int
	instance *redis.Client
	psc      *redis.PubSub
)

func Init(reset bool) {
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

	pip := instance.Pipeline()

	if reset {
		instance.FlushDB()
	}

	// Save our ingest ID
	ingestRes, _ := instance.ClientID().Result()
	ingestID = int(ingestRes)

	// Initialize jukebox
	// TODO: Make sure this works correctly when there are multiple ingests
	pip.HSet("queuestatus", utils.StructToMap(models.QueueStatus{SongEnd: time.Now()}))
	pip.Exec()
}

func GetIngestID() int {
	return ingestID
}

func GetInstance() *redis.Client {
	return instance
}

func ListenForUpdates(callback func(msg []byte)) {
	ingests, err := instance.SMembers("ingests").Result()

	if err != nil {
		panic(err)
	}

	// Let other ingest servers know about this one
	instance.Publish("ingest", strconv.Itoa(ingestID))

	// subscribe to existing ingests, send id to master
	psc = instance.Subscribe(append(ingests, "ingest")...)

	// Add this ingest to the ingests set
	instance.SAdd("ingests", ingestID)

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
	// Set our name so we can identify this client as an ingest
	cmd := redis.NewStringCmd("client", "setname", ingestClientName)
	instance.Process(cmd)

	for range time.NewTicker(time.Second).C {
		// Get list of clients connected to Redis
		clientsRes, _ := instance.ClientList().Result()

		// The leader is the first client -- the oldest connection
		clients := strings.Split(clientsRes, "\n")

		var leaderID int
		foundLeader := false

		connectedIngestIDs := mapset.NewSet()

		for _, client := range clients {
			clientParts := strings.Split(client, " ")

			if len(clientParts) < 4 {
				// Probably a newline, invalid client
				continue
			}

			clientName := strings.Split(clientParts[3], "=")[1]

			if clientName != ingestClientName {
				// This redis client is something else, probably redis-cli
				continue
			}

			clientID := strings.Split(clientParts[0], "=")[1]
			connectedIngestIDs.Add(clientID)

			// Only save the leader ID for the first ingest server
			if foundLeader {
				continue
			}

			leaderID, _ = strconv.Atoi(clientID)
			foundLeader = true
		}

		// If we're not the leader, don't do any leader actions
		if leaderID != ingestID {
			continue
		}

		//////////////////////////////////////////////
		// ALL CODE BELOW IS ONLY RUN ON THE LEADER //
		//////////////////////////////////////////////

		// Take care of ingest servers that got disconnected
		savedIngestIDs, _ := instance.SMembers("ingests").Result()

		for _, id := range savedIngestIDs {
			if connectedIngestIDs.Contains(id) {
				// This ingest is currently connected to Redis
				continue
			}

			// Remove characters connected to this ingest from their rooms
			characters, _ := instance.SMembers("ingest:" + id + ":characters").Result()

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
			instance.SRem("ingests", id)
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
				songID, _ := instance.LPop("songs").Result()

				songRes, _ := instance.HGetAll("song:" + songID).Result()
				var song models.Song
				utils.Bind(songRes, &song)

				// Update queue status to reflect new song
				endTime := time.Now().Add(time.Second * time.Duration(song.Duration))
				newStatus := models.QueueStatus{endTime}
				instance.HSet("queuestatus", utils.StructToMap(newStatus))

				// Send song packet to ingests
				// p := packet.SongPacket{Song: song, Playing: true, EndTime: endTime}
				// h.Send(p)
			}
		}
	}
}

func Publish(msg encoding.BinaryMarshaler) {
	instance.Publish(strconv.Itoa(ingestID), msg)
}
