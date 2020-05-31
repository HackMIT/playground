package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/labstack/echo/v4"
)

type RoomController struct {}

type CreateRoomBody struct {
	Slug string `json:"slug"`
	Background string `json:"background"`
}

func (r RoomController) CreateRoom(c echo.Context) error {
	json := new(CreateRoomBody)

	if err := c.Bind(json); err != nil {
		panic(err)
	}

	room := models.NewRoom("background.png", json.Slug)
	roomJSON, err := db.Rh.JSONSet("rooms:" + json.Slug, ".", room)

	if err != nil {
		panic(err)
	}

	return c.JSON(http.StatusOK, roomJSON)
}

func (r RoomController) GetRooms(c echo.Context) error {
	// TODO: Error handling
	roomNames, _ := db.Instance.Keys("rooms:*").Result()

	rooms := make([]models.Room, len(roomNames))

	for i, name := range roomNames {
		roomData, _ := db.Rh.JSONGet(name, ".")

		var room models.Room
		json.Unmarshal([]byte(roomData.([]uint8)), &room)

		rooms[i] = room
	}

	return c.JSON(http.StatusOK, rooms)
}
