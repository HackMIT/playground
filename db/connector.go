package db

import (
	"github.com/nitishm/go-rejson"
	"github.com/go-redis/redis"
)

var (
	Instance *redis.Client
	Rh *rejson.Handler
)

func Init() {
	Rh = rejson.NewReJSONHandler()

	Instance = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})

	Rh.SetGoRedisClient(Instance)
}

func ListenForUpdates(callback func(msg []byte)) {
	// Listen for updates
	// TODO: Think about subscribing to channels corresponding with other
	// ingest servers, but don't subscribe to our own, and send out events
	// from this server when they are first published
	psc := Instance.Subscribe("room")

	for {
		msg, err := psc.ReceiveMessage()

		if err != nil {
			panic(err)
		}

		if msg.Channel != "room" {
			// Right now we only receive room updates
			continue
		}

		callback([]byte(msg.Payload))
	}
}
