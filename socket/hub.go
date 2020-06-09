package socket

import (
	"encoding"
	"encoding/json"
	"log"
	"strconv"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
	"github.com/techx/playground/socket/packet"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[string]*Client

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
	h.clients = map[string]*Client{}
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
			delete(h.clients, client.id)
			close(client.send)

			// Remove this client from the room
			res, _ := db.GetRejsonHandler().JSONGet("character:" + client.name, "room")
			roomBytes := res.([]byte)
			room := string(roomBytes[1:len(roomBytes) - 1])
			db.GetRejsonHandler().JSONDel("room:" + room, "characters[\"" + client.name + "\"]")

			// Notify others that this client left
			packet := new(packet.LeavePacket).Init(client.id, client.name, room)
			h.Send(room, packet)
		case message := <-h.broadcast:
			// Process incoming messages from clients
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
		client := h.clients[id]

		if client.room != room {
			continue
		}

		// TODO: If this send fails, disconnect the client
		client.send <- msg
	}
}

// Processes an incoming message from Redis
func (h *Hub) ProcessRedisMessage(msg []byte) {
	var res map[string]interface{}
	json.Unmarshal(msg, &res)

	switch res["type"] {
	case "join", "move":
		h.SendBytes(res["room"].(string), msg)
	case "teleport":
		var p packet.TeleportPacket
		json.Unmarshal(msg, &p)

		leavePacket := new(packet.LeavePacket).Init(p.Id, p.Name, p.From)
		h.Send(p.From, leavePacket)

		joinPacket := new(packet.JoinPacket).Init(p.Id, p.Name, p.To)
		h.Send(p.To, joinPacket)
	}
}

// Processes an incoming message
func (h *Hub) processMessage(m *SocketMessage) {
	res := packet.BasePacket{}

	if err := json.Unmarshal(m.msg, &res); err != nil {
		// TODO: Log to Sentry or something -- this should never happen
		log.Println("ERROR: Received invalid JSON message from", m.sender.id, "->", m.msg)
		return
	}

	switch res.Type {
	case "join":
		// Parse join packet
		res := packet.JoinPacket{}
		json.Unmarshal(m.msg, &res)

		// TODO: Replace this with some quill ID that uniquely identifies client
		m.sender.name = res.Name
		res.Id = m.sender.id

		// When a client joins, check their room and send them the relevant
		// init packet
		var character *models.Character
		characterData, err := db.GetRejsonHandler().JSONGet("character:" + m.sender.name, ".")

		var initPacket *packet.InitPacket

		if err != nil {
			// This character doesn't exist in our database, create new one
			character = new(models.Character).Init(m.sender.name, res.Name)

			// Generate init packet before new character is added to room
			initPacket = new(packet.InitPacket).Init(character.Room)

			// Add character to database
			character.Ingest = db.GetIngestID()
			db.GetRejsonHandler().JSONSet("character:" + m.sender.name, ".", character)

			// Add to room:home at (0.5, 0.5)
			key := "characters[\"" + m.sender.name + "\"]"
			db.GetRejsonHandler().JSONSet("room:home", key, character)

			// Set default position
			res.X = 0.5
			res.Y = 0.5
		} else {
			// Load character data
			json.Unmarshal(characterData.([]byte), &character)

			// Generate init packet before new character is added to room
			initPacket = new(packet.InitPacket).Init(character.Room)

			// Add to whatever room they were at
			character.Ingest = db.GetIngestID()
			key := "characters[\"" + m.sender.name + "\"]"
			db.GetRejsonHandler().JSONSet("room:" + character.Room, key, character)

			// Set position
			res.X = character.X
			res.Y = character.Y
		}

		// Send them the relevant init packet
		data, _ := initPacket.MarshalBinary()
		m.sender.send <- data

		// Add this character's id to this ingest in Redis
		db.GetInstance().SAdd("ingest:" + strconv.Itoa(character.Ingest) + ":characters", m.sender.name)

		// Send the join packet to clients and Redis
		m.sender.room = character.Room
		res.Room = character.Room

		db.Publish(res)
		h.Send(character.Room, res)
	case "move":
		// Parse move packet
		res := packet.MovePacket{}
		json.Unmarshal(m.msg, &res)
		res.Id = m.sender.id
		res.Name = m.sender.name

		// TODO: go-rejson doesn't currently support transactions, but
		// these should really all be done together

		// Get character's current room
		var character models.Character
		characterData, _ := db.GetRejsonHandler().JSONGet("character:" + m.sender.name, ".")
		json.Unmarshal(characterData.([]byte), &character)

		// Update character's position in the room
		xKey := "characters[\"" + m.sender.name + "\"][\"x\"]"
		_, err := db.GetRejsonHandler().JSONSet("room:" + character.Room, xKey, res.X)

		if err != nil {
			log.Println(err)
			log.Fatal("ERROR: Failure sending move packet to Redis")
			return
		}

		// An error here is unlikely since we just connected to Redis
		yKey := "characters[\"" + m.sender.name + "\"][\"y\"]"
		db.GetRejsonHandler().JSONSet("room:" + character.Room, yKey, res.Y)

		// Update character's position on their model
		db.GetRejsonHandler().JSONSet("character:" + character.Id, "x", res.X)
		db.GetRejsonHandler().JSONSet("character:" + character.Id, "y", res.Y)

		// Publish move event to other ingest servers
		res.Room = character.Room
		db.Publish(res)
		h.Send(character.Room, res)
	case "teleport":
		// Parse teleport packet
		res := packet.TeleportPacket{}
		json.Unmarshal(m.msg, &res)
		res.Id = m.sender.id
		res.Name = m.sender.name

		// Update this character's room
		db.GetRejsonHandler().JSONSet("character:" + m.sender.name, "room", res.To)

		// Remove this character from the previous room
		db.GetRejsonHandler().JSONDel("room:" + res.From, "characters[\"" + m.sender.name + "\"]")

		// Send them the init packet for this room
		initPacket := new(packet.InitPacket).Init(res.To)
		initPacketData, _ := initPacket.MarshalBinary()
		m.sender.send <- initPacketData
		m.sender.room = res.To

		// Add them to their new room
		var character models.Character
		data, _ := db.GetRejsonHandler().JSONGet("character:" + res.Name, ".")
		json.Unmarshal(data.([]byte), &character)

		db.GetRejsonHandler().JSONSet("room:" + res.To, "characters[\"" + m.sender.name + "\"]", character)

		// Publish event to other ingest servers
		db.Publish(res)

		resData, _ := res.MarshalBinary()
		h.ProcessRedisMessage(resData)
	}
}
