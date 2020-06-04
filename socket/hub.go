package socket

import (
	"encoding"
	"encoding/json"
	"log"
	"math"
	"time"

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
		case client := <-h.unregister:
			// When a client disconnects, remove them from our clients map
			if client := h.clients[client.id]; client.connected {
				delete(h.clients, client.id)
				close(client.send)
			}
			// Process incoming messages from clients
		case message := <-h.broadcast:
			processMessage(message)
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

// Processes an incoming message
func processMessage(m *SocketMessage) {
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

		// When a client joins, check their room and send them the relevant
		// init packet
		var character *models.Character
		characterData, err := db.GetRejsonHandler().JSONGet("character: " + res.Id, ".")

		if err != nil {
			// This character doesn't exist in our database, create new one
			character = new(models.Character).Init(m.sender.id, res.Name)
			db.GetRejsonHandler().JSONSet("character:" + res.Id, ".", character)

			// Add to room:home at (0.5, 0.5)
			key := "characters[\"" + res.Id + "\"]"
			db.GetRejsonHandler().JSONSet("room:home", key, character)
		} else {
			json.Unmarshal(characterData.([]byte), &character)
		}

		// Send them the relevant init packet
		initPacket := new(InitPacket).Init(character.Room)
		data, _ := initPacket.MarshalBinary()
		m.sender.send <- data

		// An error here is unlikely since we just connected to Redis above
		db.GetInstance().Publish("room", res).Result()
	case "move":
		// Parse move packet
		res := MovePacket{}
		json.Unmarshal(m.msg, &res)
		res.Id = m.sender.id.String()

		// TODO: go-rejson doesn't currently support transactions, but
		// these should really all be done together

		// Get character's current room
		var character models.Character
		characterData, _ := db.GetRejsonHandler().JSONGet("character:" + res.Id, ".")
		json.Unmarshal(characterData.([]byte), &character)

		// Update character's position in the room
		xKey := "characters[\"" + res.Id + "\"][\"x\"]"
		_, err := db.GetRejsonHandler().JSONSet("room:" + character.Room, xKey, res.X)

		if err != nil {
			log.Fatal("ERROR: Failure sending move packet to Redis")
			return
		}

		// An error here is unlikely since we just connected to Redis
		yKey := "characters[\"" + res.Id + "\"][\"y\"]"
		db.GetRejsonHandler().JSONSet("room:" + character.Room, yKey, res.Y)

		// Publish move event to other ingest servers
		db.GetInstance().Publish("room", res).Result()

		// Check if this character moved to a hallway
		var room models.Room
		roomData, _ := db.GetRejsonHandler().JSONGet("room:" + character.Room, ".")
		json.Unmarshal(roomData.([]byte), &room)

		for _, hallway := range room.Hallways {
			distance := math.Sqrt(math.Pow(hallway.X - res.X, 2.0) + math.Pow(hallway.Y - res.Y, 2.0))
			if distance > hallway.Radius {
				continue
			}

			// After delay, move character to different room
			// TODO: This should depend on speed, not be constant 2s
			time.AfterFunc(2 * time.Second, func() {
				changeRoomPacket := new(ChangeRoomPacket).Init(res.Id, character.Room, hallway.To)
				db.GetInstance().Publish("room", changeRoomPacket)

				// Update this character's room
				db.GetRejsonHandler().JSONSet("character:" + res.Id, "room", hallway.To)

				// Remove this character from the previous room
				db.GetRejsonHandler().JSONDel("room:" + character.Room, "characters[\"" + res.Id + "\"]")

				// Add them to their new room
				db.GetRejsonHandler().JSONSet("room:" + hallway.To, "characters[\"" + res.Id + "\"]", character)
			})

			// Make sure we only enter one hallway, in case there are
			// overlapping ones
			break
		}
	}
}
