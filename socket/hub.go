package socket

import (
	"encoding"
	"encoding/json"
	"log"
	"fmt"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/google/uuid"
)

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

func (h *Hub) Init() *Hub {
	h.broadcast = make(chan *SocketMessage)
	h.register = make(chan *Client)
	h.unregister = make(chan *Client)
	h.clients = map[uuid.UUID]*Client{}
	return h
}

// Listens for messages from websocket clients
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.id] = client

			// When a client connects, send them an init packet
			initPacket := new(InitPacket).Init("home")
			data, _ := initPacket.MarshalBinary()
			client.send <- data
		case client := <-h.unregister:
			// When a client disconnects, remove them from our clients map
			if client := h.clients[client.id]; client.connected {
				delete(h.clients, client.id)
				close(client.send)
			}
			// Process incoming messages from clients
		case message := <-h.broadcast:
			h.processMessage(message)
		}
	}
}

// Sends a message to all of our clients
func (h *Hub) Send(room string, msg encoding.BinaryMarshaler) {
	data, _ := msg.MarshalBinary()
	h.SendBytes(room, data)
}

func (h *Hub) SendBytes(room string, msg []byte) {
	for id := range h.clients {
		// TODO: Make sure room matches -- or figure out how to iterate
		// over clients without including those in different rooms
		select {
		case h.clients[id].send <- msg:
		default:
			// If the send failed, disconnect the client
			close(h.clients[id].send)
			delete(h.clients, id)
		}
	}
}

// Subscribes to a new ingest
func (h *Hub) ProcessNewIngest(msg map[string]interface {}) {
	fmt.Println("received notice of a new ingest")
	fmt.Println(msg["id"]) // i get the ID
	// something like this not sure
	// bleh := [0]string{}
	// db.GetInstance().Subscribe(bleh...)
}

// Processes an incoming message
func (h *Hub) processMessage(m *SocketMessage) {
	res := BasePacket{}

	if err := json.Unmarshal(m.msg, &res); err != nil {
		// TODO: Log to Sentry or something -- this should never happen
		log.Println("ERROR: Received invalid JSON message from", m.sender.id.String(), "->", m.msg)
		return
	}

	switch res.Type {
	case "join":
		// Parse join packet
		res := JoinPacket{}
		json.Unmarshal(m.msg, &res)
		res.Id = m.sender.id.String()

		// Add character to room in Redis
		character := new(models.Character).Init(m.sender.id, res.Name)

		key := "characters[\"" + res.Id + "\"]"
		_, err := db.GetRejsonHandler().JSONSet("room:home", key, character)

		if err != nil {
			log.Fatal("ERROR: Failure sending join packet to Redis")
			return
		}

		// An error here is unlikely since we just connected to Redis above
		db.GetInstance().Publish(db.GetIngestID(), res).Result()
		h.Send("home", res)
	case "move":
		// Parse move packet
		res := MovePacket{}
		json.Unmarshal(m.msg, &res)
		res.Id = m.sender.id.String()

		// Update character's position in the room
		// TODO: go-rejson doesn't currently support transactions, but
		// these should really be done together
		xKey := "characters[\"" + res.Id + "\"][\"x\"]"
		_, err := db.GetRejsonHandler().JSONSet("room:home", xKey, res.X)

		if err != nil {
			log.Fatal("ERROR: Failure sending move packet to Redis")
			return
		}

		// An error here is unlikely since we just connected to Redis
		yKey := "characters[\"" + res.Id + "\"][\"y\"]"
		db.GetRejsonHandler().JSONSet("room:home", yKey, res.Y)

		// Publish move event to other ingest servers
		db.GetInstance().Publish(db.GetIngestID(), res).Result()
		h.Send("home", res)
	}
}
