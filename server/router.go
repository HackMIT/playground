package server

import (
	"github.com/techx/playground/controllers"
	"github.com/gorilla/mux"
)

func newRouter() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/rooms", controllers.GetRooms).Methods("GET")
	r.HandleFunc("/rooms", controllers.CreateRoom).Methods("POST")
	return r
}
