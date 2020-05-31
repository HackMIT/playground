package controllers

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
)

type CreateRoomBody struct {
	Slug string `json:"slug"`
	Background string `json:"background"`
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	var body CreateRoomBody
	data, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(data, &body)

	room := models.NewRoom("background.png", body.Slug)

	_, err := db.Rh.JSONSet("rooms:" + body.Slug, ".", room)

	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetRooms(w http.ResponseWriter, r *http.Request) {
	// TODO: Error handling
	roomNames, _ := db.Instance.Keys("rooms:*").Result()

	rooms := make([]models.Room, len(roomNames))

	for i, name := range roomNames {
		roomData, _ := db.Rh.JSONGet(name, ".")

		var room models.Room
		json.Unmarshal([]byte(roomData.([]uint8)), &room)

		rooms[i] = room
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	roomsStr, _ := json.Marshal(rooms)
	io.WriteString(w, string(roomsStr))
}
