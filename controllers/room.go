package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"

	"github.com/labstack/echo/v4"
)

type RoomController struct {}

// POST /rooms - creates a new room
func (r RoomController) CreateRoom(c echo.Context) error {
	// Create new room model, parse JSON body
	room := new(models.Room).Init()

	if err := c.Bind(room); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid json")
	}

	// Add new room to Redis
	_, err := db.GetRejsonHandler().JSONSet("room:" + room.Slug, ".", room)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	db.GetInstance().SAdd("rooms", room.Slug)

	return c.JSON(http.StatusOK, room)
}

// GET /rooms - get all rooms
func (r RoomController) GetRooms(c echo.Context) error {
	// Get all of the room names from Redis
	roomNames, err := db.GetInstance().SMembers("rooms").Result()

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

// GET /rooms/<room_id> - get an individual room
func (r RoomController) GetRoom(c echo.Context) error {
	// Fetch this room from Redis
	var room models.Room
	roomData, _ := db.GetRejsonHandler().JSONGet("room:" + c.Param("id"), ".")
	json.Unmarshal(roomData.([]byte), &room)

	return c.JSON(http.StatusOK, room)
}

// POST /rooms/<room_id>/hallways - creates a new hallway
func (r RoomController) CreateHallway(c echo.Context) error {
	// Create new hallway model, parse JSON body
	roomId := c.Param("id")
	hallway := new(models.Hallway)

	if err := c.Bind(hallway); err != nil {
		panic(err)
	}

	// Don't allow a hallway to the same room
	if hallway.To == roomId {
		return echo.NewHTTPError(http.StatusBadRequest, "no recursive hallways allowed")
	}

	// Add new hallway to Redis
	_, err := db.GetRejsonHandler().JSONArrAppend("room:" + roomId, "hallways", hallway)

	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "room '" + roomId + "' not found")
	}

	return c.JSON(http.StatusOK, hallway)
}
