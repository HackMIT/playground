package controllers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
	"github.com/techx/playground/socket"

	"github.com/labstack/echo/v4"

	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const developerKey = "AIzaSyBbKVxrxksLlxJYno6ZG_TzHvIpXU2O3eM"

type JukeboxController struct {
	hub *socket.Hub
	client *http.Client
	service *youtube.Service
}

func (j *JukeboxController) Init(h *socket.Hub) *JukeboxController {
	j.hub = h
	// Create YouTube client
	j.client = &http.Client{
		Transport: &transport.APIKey{Key: developerKey},
	}
	var err error
	j.service, err = youtube.New(j.client)
	if err != nil {
		log.Fatalf("Error creating new YouTube client: %v", err)
	}
	return j
}

// POST /jukebox/songs - queues up a new song
func (j JukeboxController) QueueSong(c echo.Context) error {
	// Create a new song model, parse JSON body
	song := new(models.Song).Init()

	if err := c.Bind(song); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Make the YouTube API call
	call := j.service.Videos.List("snippet,contentDetails").
			Id(song.VidCode)
	response, err := call.Do()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to fetch YouTube data")
	}

	// Should only have one video
	for _, video := range response.Items {
		// Parse duration string
		duration := video.ContentDetails.Duration
		minIndex := strings.Index(duration, "M")
		secIndex := strings.Index(duration, "S")
		// Convert duration to seconds
		minutes, err := strconv.Atoi(duration[2:minIndex])
		seconds, err := strconv.Atoi(duration[minIndex + 1:secIndex])
		// Error parsing duration string
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "failed to parse video duration")
		}
		song.Duration = (minutes * 60) + seconds
		song.Title = video.Snippet.Title
		song.ThumbnailURL = video.Snippet.Thumbnails.Default.Url
	}

	_, err = db.GetRejsonHandler().JSONArrAppend("songs", ".", song)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError,
		                         "database error")
	}

	packet := new(socket.SongPacket).Init(song)
	j.hub.Send("home", packet)

	return c.JSON(http.StatusOK, song)
}
