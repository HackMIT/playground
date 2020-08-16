package db

import (
	"encoding"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/techx/playground/config"
	"github.com/techx/playground/models"

	mapset "github.com/deckarep/golang-set"
	"github.com/nitishm/go-rejson"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

const ingestClientName string = "ingest"

var (
	ingestID int
	instance *redis.Client
	psc *redis.PubSub
	rh *rejson.Handler
)

func Init(reset bool) {
	config := config.GetConfig()
	rh = rejson.NewReJSONHandler()

	instance = redis.NewClient(&redis.Options{
		Addr: config.GetString("db.addr"),
		Password: config.GetString("db.password"),
		DB: config.GetInt("db.db"),
	})

	rh.SetGoRedisClient(instance)

	if reset {
		instance.FlushDB()

		home := new(models.Room).Init()
		home.Elements = map[string]*models.Element{
			uuid.New().String(): &models.Element{
				X: 0.2,
				Y: 0.2,
				Width: 0.1,
				Path: "lamp.svg",
			},
		}
		home.Slug = "home"
		home.Hallways = map[string]*models.Hallway{
			uuid.New().String(): &models.Hallway{
				X: 0.62,
				Y: 0.59,
				Radius: 0.1,
				To: "sponsor",
			},
		}
		rh.JSONSet("room:home", ".", home)
		instance.SAdd("rooms", "home")

		sponsor := new(models.Room).Init()
		sponsor.Slug = "sponsor"
		sponsor.Sponsor = true
		sponsor.Hallways = map[string]*models.Hallway{
			uuid.New().String(): &models.Hallway{
				X: 0.03,
				Y: 0.68,
				Radius: 0.05,
				To: "home",
			},
		}
		rh.JSONSet("room:sponsor", ".", sponsor)
		instance.SAdd("rooms", "sponsor")
	}

	// Save our ingest ID
	ingestRes, _ := instance.ClientID().Result()
	ingestID = int(ingestRes)

	// Initialize jukebox
	rh.JSONSet("songs", ".", []string{})
	rh.JSONSet("queuestatus", ".", models.QueueStatus{SongEnd: time.Now()})
}

func GetIngestID() int {
	return ingestID
}

func GetInstance() *redis.Client {
	return instance
}

func GetRejsonHandler() *rejson.Handler {
	return rh
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

			for _, characterName := range characters {
				res, _ := rh.JSONGet("character:" + characterName, "room")
				room := string(res.([]byte)[1:len(res.([]byte))-1])

				rh.JSONDel("room:" + room, "characters[\"" + characterName + "\"]")
			}

			// Ingest has been taken care of, remove from set
			instance.SRem("ingests", id)
		}

		// Get song queue status
		queueStatusData, _ := rh.JSONGet("queuestatus", ".")
		var queueStatus models.QueueStatus
		json.Unmarshal(queueStatusData.([]byte), &queueStatus)
		songEnd := queueStatus.SongEnd

		// If current song ended, start next song (if there is one)
		if songEnd.Before(time.Now()) {
			queueLengthData, _ := rh.JSONArrLen("songs", ".")
			queueLength := queueLengthData.(int64)
			if queueLength > 0 {
				// Pop the next song off the queue
				songData, _ := rh.JSONArrPop("songs", ".", 0)
				var song models.Song
				json.Unmarshal(songData.([]byte), &song)
				// Update queue status to reflect new song
				newStatus := models.QueueStatus{time.Now().Add(time.Second * time.Duration(song.Duration))}
				rh.JSONSet("queuestatus", ".", newStatus)
			}
		}
	}
}

func Publish(msg encoding.BinaryMarshaler) {
	instance.Publish(strconv.Itoa(ingestID), msg)
}
