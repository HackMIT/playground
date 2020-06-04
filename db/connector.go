package db

import (
	"strconv"
	"strings"
	"time"

	"github.com/nitishm/go-rejson"
	"github.com/go-redis/redis"

	"github.com/techx/playground/config"
)

var (
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

	rh.JSONSet("songs", ".", []string{})
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
		clients, _ := instance.ClientList().Result()

		// The leader is the first client -- the oldest connection
		leader := strings.Split(clients, "\n")[0]
		leaderParts := strings.Split(leader, " ")
		leaderID, _ := strconv.Atoi(strings.Split(leaderParts[0], "=")[1])
		ingestID, _ := instance.ClientID().Result()

		// Add one because rejson creates a second client
		if leaderID + 1 != int(ingestID) {
			continue
		}

		// TODO: (#2) Take care of song ended packets here
	}
}
