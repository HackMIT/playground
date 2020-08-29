package server

import (
	"fmt"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/socket"
)

func Init(port string) {
	hub := new(socket.Hub).Init()

	// Wait for socket messages
	go hub.Run()

	// Listen for events from other ingest servers
	go db.ListenForUpdates(hub.ProcessRedisMessage)

	// Check if we're the leader and do things if so
	go db.MonitorLeader()

	// Websocket connection endpoint
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		socket.ServeWs(hub, w, r)
	})

	// REST endpoints
	r := newRouter(hub)
	http.Handle("/", r)

	addr := ":" + port

	// Start the server
	fmt.Println("Serving at", addr)
	err := http.ListenAndServe(addr, nil)

	if err != nil {
		panic(err)
	}
}
