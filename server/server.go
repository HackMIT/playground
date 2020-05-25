package server

import (
	"fmt"
	"net/http"

	"github.com/techx/playground/config"
	"github.com/techx/playground/world"
)

func Init() {
	hub := world.NewHub()
	wo := world.NewWorld()
	go hub.Run(wo)

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		world.ServeWs(hub, w, r)
	})

	r := newRouter()
	http.Handle("/", r)

	config := config.GetConfig()

	fmt.Println("Serving at", config.GetString("server.addr"))
	err := http.ListenAndServe(config.GetString("server.addr"), nil)

	if err != nil {
		panic(err)
	}
}
