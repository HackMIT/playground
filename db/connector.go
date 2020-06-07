package db

import (
	"strconv"
	"strings"
	"time"

	"github.com/techx/playground/config"

	mapset "github.com/deckarep/golang-set"
	"github.com/nitishm/go-rejson"
	"github.com/go-redis/redis"
)

const ingestClientName string = "ingest"

var (
	ingestID int
	instance *redis.Client
	rh *rejson.Handler
)

func Init() {
	config := config.GetConfig()
	rh = rejson.NewReJSONHandler()

	instance = redis.NewClient(&redis.Options{
		Addr: config.GetString("db.addr"),
		Password: config.GetString("db.password"),
		DB: config.GetInt("db.db"),
	})

	rh.SetGoRedisClient(instance)

	// Save our ingest ID
	ingestRes, _ := instance.ClientID().Result()
	ingestID = int(ingestRes)
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
	// TODO (#1): Think about subscribing to channels corresponding with other
	// ingest servers, but don't subscribe to our own, and send out events
	// from this server when they are first published
	psc := instance.Subscribe("room")

	for {
		msg, err := psc.ReceiveMessage()

		if err != nil {
			// Stop server if we disconnect from Redis
			panic(err)
		}

		if msg.Channel != "room" {
			// Right now we only receive room updates
			continue
		}

		callback([]byte(msg.Payload))
	}
}

func MonitorLeader() {
	for range time.NewTicker(time.Second).C {
		// Get list of clients connected to Redis
		clientsRes, _ := instance.ClientList().Result()

		// The leader is the first client -- the oldest connection
		clients := strings.Split(clientsRes, "\n")

		var leaderID int
		foundLeader := false

		ingestIDs := mapset.NewSet()

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

			ingestID, _ = strconv.Atoi(strings.Split(clientParts[0], "=")[1])
			ingestIDs.Add(ingestID)

			// Only save the leader ID for the first ingest server
			if foundLeader {
				continue
			}

			leaderID = ingestID
			foundLeader = true
		}

		// If we're not the leader, don't do any leader actions
		if leaderID != ingestID {
			continue
		}

		// TODO: (#2) Take care of song ended packets here

		// TODO: Remove clients who were connected to ingest servers that are no
		// longer online from their rooms
	}
}
