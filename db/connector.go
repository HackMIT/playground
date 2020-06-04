package db

import (
	"github.com/techx/playground/config"

	"github.com/nitishm/go-rejson"
	"github.com/go-redis/redis"
	"github.com/google/uuid"
)

var (
	ingestID string
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

	// generate new id
	id, _ := uuid.NewUUID()
	ingestID = id.String()
}

func GetIngestID() string {
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
	ingests, err := instance.SMembers("ingests").Result()

	if err != nil {
		panic(err)
	}

	// subscribe to existing ingests
	psc := instance.Subscribe(ingests...)
	instance.SAdd("ingests", ingestID)

	for {
		msg, err := psc.ReceiveMessage()

		if err != nil {
			// Stop server if we disconnect from Redis
			panic(err)
		}

		// if msg.Channel != "room" {
		// 	// Right now we only receive room updates
		// 	continue
		// }

		callback([]byte(msg.Payload))
	}
}
