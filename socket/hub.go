package socket

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
	"github.com/techx/playground/socket/packet"

	"github.com/dgrijalva/jwt-go"
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

			if client.character == nil {
				continue
			}

			// Remove this client from the room
			res, _ := db.GetRejsonHandler().JSONGet("character:" + client.character.ID, "room")
			roomBytes := res.([]byte)
			room := string(roomBytes[1:len(roomBytes) - 1])
			db.GetRejsonHandler().JSONDel("room:" + room, "characters[\"" + client.character.ID + "\"]")

			// Notify others that this client left
			packet := packet.NewLeavePacket(client.character, room)
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

		if client.character == nil {
			continue
		}

		if client.character.Room != room {
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

		leavePacket := packet.NewLeavePacket(p.Character, p.From)
		h.Send(p.From, leavePacket)

		joinPacket := packet.NewJoinPacket(p.Character)
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
	case "chat":
		res := packet.ChatPacket{}
		json.Unmarshal(m.msg, &res)

		// Publish chat event to other ingest servers
		res.Room = m.sender.character.Room
		res.ID = m.sender.character.ID

		db.Publish(res)
		h.Send(res.Room, res)
	case "join":
		// Parse join packet
		res := packet.JoinPacket{}
		json.Unmarshal(m.msg, &res)

		var character *models.Character
		var characterID string
		var initPacket *packet.InitPacket

		if res.Name != "" {
			character = models.NewCharacter(res.Name)
			characterID = character.ID

			// Add character to database
			character.Ingest = db.GetIngestID()
			db.GetRejsonHandler().JSONSet("character:" + characterID, ".", character)

			// Generate init packet before new character is added to room
			initPacket = packet.NewInitPacket(characterID, character.Room, true)

			// Add to room:home at (0.5, 0.5)
			key := "characters[\"" + characterID + "\"]"
			db.GetRejsonHandler().JSONSet("room:home", key, character)
		} else if res.QuillToken != "" {
			// Fetch data from Quill
			quillValues := map[string]string{
				"token": res.QuillToken,
			}

			quillBody, _ := json.Marshal(quillValues)
			// TODO: Error handling
			resp, _ := http.Post("https://my.hackmit.org/auth/sso/exchange", "application/json", bytes.NewBuffer(quillBody))

			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)

			var quillData map[string]interface{}
			err := json.Unmarshal(body, &quillData)

			if err != nil {
				// Likely invalid SSO token
				return
			}

			// Load this client's character
			characterID, err = db.GetInstance().HGet("quillToCharacter", quillData["id"].(string)).Result()

			if err != nil {
				// Never seen this character before, create a new one
				character = models.NewCharacterFromQuill(quillData)
				characterID = character.ID

				// Add character to database
				character.Ingest = db.GetIngestID()
				db.GetRejsonHandler().JSONSet("character:" + characterID, ".", character)
				db.GetInstance().HSet("quillToCharacter", quillData["id"].(string), characterID)

				// Generate init packet before new character is added to room
				initPacket = packet.NewInitPacket(characterID, character.Room, true)

				// Add to room:home at (0.5, 0.5)
				key := "characters[\"" + characterID + "\"]"
				db.GetRejsonHandler().JSONSet("room:home", key, character)
			} else {
				// This person has logged in before, fetch from Redis
				characterData, _ := db.GetRejsonHandler().JSONGet("character:" + characterID, ".")
				json.Unmarshal(characterData.([]byte), &character)

				// Generate init packet before new character is added to room
				initPacket = packet.NewInitPacket(characterID, character.Room, true)

				// Add to whatever room they were at
				character.Ingest = db.GetIngestID()
				key := "characters[\"" + characterID + "\"]"
				db.GetRejsonHandler().JSONSet("room:" + character.Room, key, character)
			}
		} else if res.Token != "" {
			// TODO: Error handling
			token, err := jwt.Parse(res.Token, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}

				config := config.GetConfig()
				return []byte(config.GetString("jwt.secret")), nil
			})

			if err != nil {
				errorPacket := packet.NewErrorPacket(1)
				data, _ := json.Marshal(errorPacket)
				m.sender.send <- data
				return
			}

			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				characterID = claims["id"].(string)
			} else {
				// TODO: Error handling
				return
			}

			// This person has logged in before, fetch from Redis
			characterData, err := db.GetRejsonHandler().JSONGet("character:" + characterID, ".")

			if err != nil {
				errorPacket := packet.NewErrorPacket(1)
				data, _ := json.Marshal(errorPacket)
				m.sender.send <- data
				return
			}

			json.Unmarshal(characterData.([]byte), &character)

			// Generate init packet before new character is added to room
			initPacket = packet.NewInitPacket(characterID, character.Room, true)

			// Add to whatever room they were at
			character.Ingest = db.GetIngestID()
			key := "characters[\"" + characterID + "\"]"
			db.GetRejsonHandler().JSONSet("room:" + character.Room, key, character)
		} else {
			// Client provided no authentication
			return
		}

		// Make sure SSO token is omitted from join packet that is sent to clients
		res.Name = ""
		res.QuillToken = ""
		res.Token = ""

		// Send them the relevant init packet
		data, _ := initPacket.MarshalBinary()
		m.sender.send <- data

		// Add this character's id to this ingest in Redis
		db.GetInstance().SAdd("ingest:" + strconv.Itoa(character.Ingest) + ":characters", characterID)

		// Send the join packet to clients and Redis
		m.sender.character = character
		res.Character = character

		db.Publish(res)
		h.Send(character.Room, res)
	case "move":
		if m.sender.character == nil {
			return
		}

		// Parse move packet
		res := packet.MovePacket{}
		json.Unmarshal(m.msg, &res)

		// TODO: go-rejson doesn't currently support transactions, but
		// these should really all be done together

		// Update character's position in the room
		xKey := "characters[\"" + m.sender.character.ID + "\"][\"x\"]"
		_, err := db.GetRejsonHandler().JSONSet("room:" + m.sender.character.Room, xKey, res.X)

		if err != nil {
			log.Println(err)
			log.Fatal("ERROR: Failure sending move packet to Redis")
			return
		}

		// An error here is unlikely since we just connected to Redis
		yKey := "characters[\"" + m.sender.character.ID + "\"][\"y\"]"
		db.GetRejsonHandler().JSONSet("room:" + m.sender.character.Room, yKey, res.Y)

		// Update character's position on their model
		db.GetRejsonHandler().JSONSet("character:" + m.sender.character.ID, "x", res.X)
		db.GetRejsonHandler().JSONSet("character:" + m.sender.character.ID, "y", res.Y)

		// Publish move event to other ingest servers
		res.Room = m.sender.character.Room
		res.ID = m.sender.character.ID

		db.Publish(res)
		h.Send(m.sender.character.Room, res)
	case "teleport":
		// Parse teleport packet
		res := packet.TeleportPacket{}
		json.Unmarshal(m.msg, &res)

		// Update this character's room
		db.GetRejsonHandler().JSONSet("character:" + m.sender.character.ID, "room", res.To)
		db.GetRejsonHandler().JSONSet("character:" + m.sender.character.ID, "x", 0.5)
		db.GetRejsonHandler().JSONSet("character:" + m.sender.character.ID, "y", 0.5)

		// Remove this character from the previous room
		db.GetRejsonHandler().JSONDel("room:" + res.From, "characters[\"" + m.sender.character.ID + "\"]")

		// Send them the init packet for this room
		initPacket := packet.NewInitPacket(m.sender.character.ID, res.To, false)
		initPacketData, _ := initPacket.MarshalBinary()
		m.sender.send <- initPacketData
		m.sender.character.Room = res.To

		// Add them to their new room
		var character models.Character
		data, _ := db.GetRejsonHandler().JSONGet("character:" + m.sender.character.ID, ".")
		json.Unmarshal(data.([]byte), &character)
		db.GetRejsonHandler().JSONSet("room:" + res.To, "characters[\"" + m.sender.character.ID + "\"]", character)

		// Publish event to other ingest servers
		res.Character = &character
		db.Publish(res)

		resData, _ := res.MarshalBinary()
		h.ProcessRedisMessage(resData)
	case "sponsor":
		fmt.Println("Sponsor packet received!")
	case "hackerqueue":
		fmt.Println("Hackerqueue packet received!")
	}
}
