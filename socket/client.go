package socket

import (
	"github.com/techx/playground/db/models"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	// ID uniquely identifying this client
	id string

	// This client's character
	character *models.Character
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	c := new(Client)
	c.hub = hub
	c.conn = conn
	c.send = make(chan []byte, 4096)
	c.id = uuid.New().String()
	return c
}
