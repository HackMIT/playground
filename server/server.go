package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/techx/playground/db"
	"github.com/techx/playground/socket"
)

func Init(port int) {
	hub := new(socket.Hub).Init()

	// Wait for socket messages
	go hub.Run()

	// Listen for events from other ingest servers
	go db.ListenForUpdates(func(data []byte) {
		var msg map[string]interface{}
		json.Unmarshal(data, &msg)

		switch msg["type"] {
		case "join", "move":
			hub.SendBytes("home", data)
		}
	})

	// Check if we're the leader and do things if so
	go db.MonitorLeader()

	// Websocket connection endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket.ServeWs(hub, w, r)
	})

	// REST endpoints
	r := newRouter()
	http.Handle("/", r)

	addr := ":" + strconv.Itoa(port)

	// Start the server
	fmt.Println("Serving at", addr)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		panic(err)
	}
}
