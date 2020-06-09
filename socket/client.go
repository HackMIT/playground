package socket

import (
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

	// The name of the client
	name string

	// The room this client is in
	room string

	// True if this client is not authenticated
	anonymous bool
}

func (c *Client) Init(hub *Hub, conn *websocket.Conn) *Client {
	c.hub = hub
	c.conn = conn
	c.send = make(chan []byte, 256)

	clientID, _ := uuid.NewUUID()
	c.id = clientID.String()

	c.anonymous = true

	return c
}
