package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/go-redis/redis/v7"
)

var addr = flag.String("addr", ":8080", "http service address")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	// if r.URL.Path != "/" {
	// 	http.Error(w, "Not found", http.StatusNotFound)
	// 	return
	// }
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.String()[1:]

	if (path == "") {
		path = "home.html"
	}

	http.ServeFile(w, r, path)
}

func main() {
	flag.Parse()
	db := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})
	write_err := db.Set("key", "value", 0).Err()

	if write_err != nil {
		panic(write_err)
	}

	val, _ := db.Get("key").Result()
	log.Println(val)

	hub := newHub()
	world := newWorld()
	go hub.run(world)
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
