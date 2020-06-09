package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/socket"
)

func Init() {
	hub := new(socket.Hub).Init()

	// Wait for socket messages
	go hub.Run()

	// Listen for events from other ingest servers
	go db.ListenForUpdates(func(data []byte) {
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)

		fmt.Println("received something in server")

		switch msg["type"] {
		case "join", "move":
			hub.SendBytes("home", data)
		default:
			hub.ProcessNewIngest(msg)
		}	
	})

	// Notify others ingests of joining this runs
	hub.NotifyNewIngest()
	
	// Websocket connection endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket.ServeWs(hub, w, r)
	})

	// REST endpoints
	r := newRouter()
	http.Handle("/", r)

	config := config.GetConfig()
	addr := config.GetString("server.addr")

	// Start the server
	fmt.Println("Serving at", addr)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		panic(err)
	}
}
