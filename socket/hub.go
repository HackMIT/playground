package socket

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
    "hash/fnv"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
    "strings"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
	"github.com/techx/playground/socket/packet"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"google.golang.org/api/googleapi/transport"
	"google.golang.org/api/youtube/v3"
)

const youtubeAPIKey = "AIzaSyBbKVxrxksLlxJYno6ZG_TzHvIpXU2O3eM"

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

        if room == "*" {
            client.send <- msg
            continue
        }

        if client.character.Room == room {
            client.send <- msg
            continue
        }

        if strings.Contains(room, "character:") && client.character.ID == strings.Split(room, ":")[1] {
            client.send <- msg
            continue
        }

		// TODO: If this send fails, disconnect the client
	}
}

// Processes an incoming message from Redis
func (h *Hub) ProcessRedisMessage(msg []byte) {
	var res map[string]interface{}
	json.Unmarshal(msg, &res)

	switch res["type"] {
    case "message":
        h.SendBytes("character:" + res["recipient"].(string), msg)

        if res["recipient"].(string) != res["sender"].(string) {
            h.SendBytes("character:" + res["sender"].(string), msg)
        }
	case "join", "move":
		h.SendBytes(res["room"].(string), msg)
	case "element_add", "element_delete", "element_update":
		h.SendBytes(res["slug"].(string), msg)
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
	case "element_add":
		res := packet.ElementAddPacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		res.ID = uuid.New().String()
		db.GetRejsonHandler().JSONSet("room:" + res.Room, "elements[\"" + res.ID+ "\"]", res.Element)

		// Publish event to other ingest servers
		db.Publish(res)
		h.Send(res.Room, res)
	case "element_delete":
		res := packet.ElementDeletePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		db.GetRejsonHandler().JSONDel("room:" + res.Room, "elements[\"" + res.ID + "\"]")

		// Publish event to other ingest servers
		db.Publish(res)
		h.Send(res.Room, res)
	case "element_update":
		res := packet.ElementUpdatePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		db.GetRejsonHandler().JSONSet("room:" + res.Room, "elements[\"" + res.ID + "\"]", res.Element)

		// Publish event to other ingest servers
		db.Publish(res)
		h.Send(res.Room, res)
	case "hallway_add":
		res := packet.HallwayAddPacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		res.ID = uuid.New().String()
		db.GetRejsonHandler().JSONSet("room:" + res.Room, "hallways[\"" + res.ID + "\"]", res.Hallway)

		// Publish event to other ingest servers
		db.Publish(res)
		h.Send(res.Room, res)
	case "hallway_delete":
		res := packet.HallwayDeletePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		db.GetRejsonHandler().JSONDel("room:" + res.Room, "hallways[\"" + res.ID + "\"]")

		// Publish event to other ingest servers
		db.Publish(res)
		h.Send(res.Room, res)
	case "hallway_update":
		res := packet.HallwayUpdatePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		db.GetRejsonHandler().JSONSet("room:" + res.Room, "hallways[\"" + res.ID + "\"]", res.Hallway)

		// Publish event to other ingest servers
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
    case "message":
        // Parse message packet
        res := packet.MessagePacket{}
        json.Unmarshal(m.msg, &res)
        res.Sender = m.sender.character.ID

        // Save this message
        messageID := uuid.New().String()
        db.GetRejsonHandler().JSONSet("message:" + messageID, ".", res)

        ha := fnv.New32a()
        ha.Write([]byte(res.Sender))
        senderHash := ha.Sum32()

        ha.Reset()
        ha.Write([]byte(res.Recipient))
        recipientHash := ha.Sum32()

        conversationKey := "conversation:" + res.Sender + ":" + res.Recipient

        if recipientHash < senderHash {
            conversationKey = "conversation:" + res.Recipient + ":" + res.Sender
        }

        db.GetInstance().RPush(conversationKey, messageID)

        db.Publish(res)
        h.Send("character:" + res.Recipient, res)

        if res.Recipient != res.Sender {
            h.Send("character:" + res.Sender, res)
        }
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
    case "song":
        // Parse song packet
        res := packet.SongPacket{}
        json.Unmarshal(m.msg, &res)

        // Make the YouTube API call
        youtubeClient, _ := youtube.New(&http.Client{
            Transport: &transport.APIKey{Key: youtubeAPIKey},
        })
        call := youtubeClient.Videos.List("snippet,contentDetails").
                Id(res.VidCode)

        response, err := call.Do()
        if err != nil {
            // TODO: Send error packet
            panic(err)
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
                // TODO: Send error packet
                panic(err)
            }

            res.Duration = (minutes * 60) + seconds
            res.Title = video.Snippet.Title
            res.ThumbnailURL = video.Snippet.Thumbnails.Default.Url
        }

        _, err = db.GetRejsonHandler().JSONArrAppend("songs", ".", res.Song)

        if err != nil {
            // TODO: Send error packet
            panic(err)
        }

        db.Publish(res)
        h.Send("*", res)
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
	}
}
