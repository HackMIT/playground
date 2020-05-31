package socket

import (
	"encoding/json"
	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
	"github.com/google/uuid"
)

// SocketMessage stores the message sent over WS and the client who sent it
type SocketMessage struct {
	msg []byte
	sender *Client
}

func (m SocketMessage) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m SocketMessage) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[uuid.UUID]*Client

	// Inbound messages from the clients.
	broadcast chan *SocketMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan *SocketMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[uuid.UUID]*Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.id] = client

			data, _ := json.Marshal(newInitPacket())
			client.send <- data
		case client := <-h.unregister:
			if client := h.clients[client.id]; client.connected {
				delete(h.clients, client.id)
				close(client.send)
			}
		case message := <-h.broadcast:
			processMessage(message)
		}
	}
}

func (h *Hub) Send(room string, msg []byte) {
	for id := range h.clients {
		// TODO: Make sure room matches -- or figure out how to iterate
		// over clients without including those in different rooms
		select {
		case h.clients[id].send <- msg:
		default:
			close(h.clients[id].send)
			delete(h.clients, id)
		}
	}
}

func processMessage(m *SocketMessage) {
	res := BasePacket{}

	if err := json.Unmarshal(m.msg, &res); err != nil {
		// TODO: Better error handling
		panic(err)
	}

	switch res.Type {
	case "join":
		res := JoinPacket{}

		if err := json.Unmarshal(m.msg, &res); err != nil {
			panic(err)
		}

		res.Id = m.sender.id.String()

		character := models.NewCharacter(m.sender.id, res.Name)

		_, err := db.Rh.JSONSet("rooms:home", "characters[\"" + m.sender.id.String() + "\"]", character)

		if err != nil {
			panic(err)
		}

		_, publishErr := db.Instance.Publish("room", res).Result()

		if publishErr != nil {
			panic(publishErr)
		}
	case "move":
		res := MovePacket{}

		if err := json.Unmarshal(m.msg, &res); err != nil {
			panic(err)
		}

		res.Id = m.sender.id.String()

		// TODO: go-rejson doesn't currently support transactions, but
		// these should really be done together
		db.Rh.JSONSet("rooms:home", "characters[\"" + m.sender.id.String() + "\"][\"x\"]", res.X)
		db.Rh.JSONSet("rooms:home", "characters[\"" + m.sender.id.String() + "\"][\"y\"]", res.Y)

		_, publishErr := db.Instance.Publish("room", res).Result()

		if publishErr != nil {
			panic(publishErr)
		}
	}
}
