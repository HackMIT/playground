package controllers

import (
	"fmt"
	"net/http"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
	"github.com/techx/playground/socket"

	"github.com/labstack/echo/v4"
)

type JukeboxController struct {
	hub *socket.Hub
}

func (j *JukeboxController) Init(h *socket.Hub) *JukeboxController {
	j.hub = h
	return j
}

// POST /jukebox/songs - queues up a new song
func (j JukeboxController) QueueSong(c echo.Context) error {
	// Create a new song model, parse JSON body
	song := new(models.Song).Init()

	if err := c.Bind(song); err != nil {
		panic(err)
	}

	_, err := db.GetRejsonHandler().JSONArrAppend("songs", ".", song)

	if err != nil {
		fmt.Println(err)
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	packet := new(socket.SongPacket).Init(song)
	j.hub.Send("home", packet)

	return c.JSON(http.StatusOK, song)
}
