package controllers

import (
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"

	"github.com/go-redis/redis/v7"
	"github.com/labstack/echo/v4"
)

type RoomController struct{}

// GET /rooms - get all rooms
func (r RoomController) GetRooms(c echo.Context) error {
	// Get all of the room names from Redis
	roomNames, err := db.GetInstance().SMembers("rooms").Result()

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
			"database error")
	}

	// Load each room into this array
	pip := db.GetInstance().Pipeline()
	roomCmds := make([]*redis.StringStringMapCmd, len(roomNames))

	for i, name := range roomNames {
		roomCmds[i] = pip.HGetAll("room:" + name)
	}

	pip.Exec()
	rooms := make([]models.Room, len(roomNames))

	for i, roomCmd := range roomCmds {
		roomData, _ := roomCmd.Result()
		utils.Bind(roomData, &rooms[i])
	}

	return c.JSON(http.StatusOK, rooms)
}
