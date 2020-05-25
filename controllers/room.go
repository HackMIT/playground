package controllers

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/mitchellh/mapstructure"
)

type CreateRoomBody struct {
	Slug string `json:"slug"`
	Background string `json:"background"`
}

func CreateRoom(w http.ResponseWriter, r *http.Request) {
	var body CreateRoomBody
	data, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(data, &body)

	_, err := db.Instance.HMSet("rooms:" + body.Slug, map[string]interface{}{
		"background": body.Background,
	}).Result()

	if err != nil {
		panic(err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func GetRooms(w http.ResponseWriter, r *http.Request) {
	// TODO: Error handling
	roomNames, _ := db.Instance.Keys("rooms:*").Result()

	rooms := make(map[string]models.Room)

	for _, name := range roomNames {
		roomData, _ := db.Instance.HGetAll(name).Result()

		var room models.Room
		mapstructure.Decode(roomData, &room)

		// Remove rooms: prefix (rooms:home -> home)
		roomName := name[6:]
		rooms[roomName] = room
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	roomsStr, _ := json.Marshal(rooms)
	io.WriteString(w, string(roomsStr))
}
