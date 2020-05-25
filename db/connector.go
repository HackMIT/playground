package db

import (
	"github.com/go-redis/redis/v7"
)

var Instance = redis.NewClient(&redis.Options{
	Addr: "localhost:6379",
	Password: "",
	DB: 0,
})
