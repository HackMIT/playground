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
	psc *redis.PubSub
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
	ingests, err := instance.SMembers("ingests").Result()

	if err != nil {
		panic(err)
	}

	// Let other ingest servers know about this one
	instance.Publish("ingest", ingestID)

	// subscribe to existing ingests, send id to master
	psc = instance.Subscribe(append(ingests, "ingest")...)
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
