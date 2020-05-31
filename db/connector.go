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
