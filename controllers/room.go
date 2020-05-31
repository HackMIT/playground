package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/labstack/echo/v4"
)

type RoomController struct {}

// POST /rooms - creates a new room
func (r RoomController) CreateRoom(c echo.Context) error {
	// Create new room model, parse JSON body
	room := new(models.Room).Init()

	if err := c.Bind(room); err != nil {
		panic(err)
	}

	// Add new room to Redis
	_, err := db.GetRejsonHandler().JSONSet("room:" + room.Slug, ".", room)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	return c.JSON(http.StatusOK, room)
}

func (r RoomController) GetRooms(c echo.Context) error {
	// Get all of the room names from Redis
	roomNames, err := db.GetInstance().Keys("room:*").Result()

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	// Load each room into this array
	rooms := make([]models.Room, len(roomNames))

	for i, name := range roomNames {
		// Error here is unlikely because we already fetched from the DB
		roomData, _ := db.GetRejsonHandler().JSONGet(name, ".")
		json.Unmarshal(roomData.([]byte), &rooms[i])
	}

	return c.JSON(http.StatusOK, rooms)
}
