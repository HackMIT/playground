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
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/socket/packet"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v7"
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
			room, _ := db.GetInstance().HGet("character:"+client.character.ID, "room").Result()
			db.GetInstance().SRem("room:"+room+":characters", client.character.ID)

			// Notify others that this client left
			packet := packet.NewLeavePacket(client.character, room)
			h.Send(packet)
		case message := <-h.broadcast:
			// Process incoming messages from clients
			h.processMessage(message)
		}
	}
}

// Sends a message to all of our clients
func (h *Hub) Send(msg encoding.BinaryMarshaler) {
	// Send to other ingest servers
	db.Publish(msg)

	// Send to clients connected to this ingest
	data, _ := msg.MarshalBinary()
	h.ProcessRedisMessage(data)
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
		h.SendBytes("character:"+res["to"].(string), msg)

		if res["to"].(string) != res["from"].(string) {
			h.SendBytes("character:"+res["from"].(string), msg)
		}
	case "chat", "move", "leave":
		h.SendBytes(res["room"].(string), msg)
	case "join":
		h.SendBytes(res["character"].(map[string]interface{})["room"].(string), msg)
	case "element_add", "element_delete", "element_update", "hallway_add", "hallway_delete", "hallway_update":
		h.SendBytes(res["room"].(string), msg)
	case "song":
		h.SendBytes("*", msg)
	case "teleport", "teleport_home":
		var p packet.TeleportPacket
		json.Unmarshal(msg, &p)

		leavePacket, _ := packet.NewLeavePacket(p.Character, p.From).MarshalBinary()
		h.SendBytes(p.From, leavePacket)

		joinPacket, _ := packet.NewJoinPacket(p.Character).MarshalBinary()
		h.SendBytes(p.To, joinPacket)
	}
}

// Processes an incoming message
func (h *Hub) processMessage(m *SocketMessage) {
	res := packet.BasePacket{}

	if err := json.Unmarshal(m.msg, &res); err != nil {
		// TODO: Log to Sentry or something -- this should never happen
		fmt.Println(err)
		log.Println("ERROR: Received invalid JSON message from", m.sender.id, "->", string(m.msg))
		return
	}

	switch res.Type {
	case "chat":
		res := packet.ChatPacket{}
		json.Unmarshal(m.msg, &res)

		// Publish chat event to other clients
		res.Room = m.sender.character.Room
		res.ID = m.sender.character.ID
		h.Send(res)
	case "element_add":
		res := packet.ElementAddPacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		res.ID = uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("element:"+res.ID, db.StructToMap(res.Element))
		pip.SAdd("room:"+res.Room+":elements", res.ID)
		pip.Exec()

		// Publish event to other clients
		h.Send(res)
	case "element_delete":
		res := packet.ElementDeletePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		pip := db.GetInstance().Pipeline()
		pip.Del("element:" + res.ID)
		pip.SRem("room:"+res.Room+":elements", res.ID)
		pip.Exec()

		// Publish event to other ingest servers
		h.Send(res)
	case "element_update":
		res := packet.ElementUpdatePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		db.GetInstance().HSet("element:"+res.ID, db.StructToMap(res.Element))

		// Publish event to other ingest servers
		h.Send(res)
	case "get_map":
		// Send locations back to client
		resp := packet.NewMapPacket()
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case "get_messages":
		res := packet.GetMessagesPacket{}
		json.Unmarshal(m.msg, &res)
		sender := m.sender.character.ID

		ha := fnv.New32a()
		ha.Write([]byte(sender))
		senderHash := ha.Sum32()

		ha.Reset()
		ha.Write([]byte(res.Recipient))
		recipientHash := ha.Sum32()

		conversationKey := "conversation:" + sender + ":" + res.Recipient

		if recipientHash < senderHash {
			conversationKey = "conversation:" + res.Recipient + ":" + sender
		}

		messageIDs, _ := db.GetInstance().LRange(conversationKey, -100, -1).Result()

		pip := db.GetInstance().Pipeline()
		messageCmds := make([]*redis.StringStringMapCmd, len(messageIDs))

		for i, messageID := range messageIDs {
			messageCmds[i] = pip.HGetAll("message:" + messageID)
		}

		pip.Exec()
		messages := make([]*models.Message, len(messageIDs))

		for i, messageCmd := range messageCmds {
			messageRes, _ := messageCmd.Result()
			messages[i] = new(models.Message)
			db.Bind(messageRes, messages[i])
		}

		resp := packet.NewMessagesPacket(messages, res.Recipient)
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case "hallway_add":
		res := packet.HallwayAddPacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		res.ID = uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("hallway:"+res.ID, db.StructToMap(res.Hallway))
		pip.SAdd("room:"+res.Room+":hallways", res.ID)
		pip.Exec()

		// Publish event to other ingest servers
		h.Send(res)
	case "hallway_delete":
		res := packet.HallwayDeletePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		pip := db.GetInstance().Pipeline()
		pip.Del("hallway:" + res.ID)
		pip.SRem("room:"+res.Room+":hallways", res.ID)
		pip.Exec()

		// Publish event to other ingest servers
		h.Send(res)
	case "hallway_update":
		res := packet.HallwayUpdatePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		db.GetInstance().HSet("hallway:"+res.ID, db.StructToMap(res.Hallway))

		// Publish event to other ingest servers
		h.Send(res)
	case "join":
		// Parse join packet
		res := packet.JoinPacket{}
		json.Unmarshal(m.msg, &res)

		character := new(models.Character)
		var characterID string
		var initPacket *packet.InitPacket

		pip := db.GetInstance().Pipeline()

		if res.Name != "" {
			character = models.NewCharacter(res.Name)
			characterID = character.ID

			// Add character to database
			character.Ingest = db.GetIngestID()
			db.GetInstance().HSet("character:"+characterID, db.StructToMap(character))

			// Generate init packet before new character is added to room
			initPacket = packet.NewInitPacket(characterID, character.Room, true)

			// Add to room:home at (0.5, 0.5)
			pip.SAdd("room:home:characters", characterID)
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
				// TODO: Send error packet
				return
			}

			admitted := quillData["status"].(map[string]interface{})["admitted"].(bool)

			if !admitted {
				// Don't allow non-admitted hackers to access Playground
				// TODO: Send error packet
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
				pip.HSet("character:"+characterID, db.StructToMap(character))
				pip.HSet("quillToCharacter", quillData["id"].(string), characterID)

				// Generate init packet before new character is added to room
				initPacket = packet.NewInitPacket(characterID, character.Room, true)

				// Add to room:home at (0.5, 0.5)
				pip.SAdd("room:home:characters", characterID)
			} else {
				// This person has logged in before, fetch from Redis
				characterRes, _ := db.GetInstance().HGetAll("character:" + characterID).Result()
				db.Bind(characterRes, &character)

				// Generate init packet before new character is added to room
				initPacket = packet.NewInitPacket(characterID, character.Room, true)

				// Add to whatever room they were in
				pip.SAdd("room:"+character.Room+":characters", character.ID)
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
			characterRes, err := db.GetInstance().HGetAll("character:" + characterID).Result()

			if err != nil || len(characterRes) == 0 {
				errorPacket := packet.NewErrorPacket(1)
				data, _ := json.Marshal(errorPacket)
				m.sender.send <- data
				return
			}

			db.Bind(characterRes, character)
			character.ID = characterID

			// Generate init packet before new character is added to room
			initPacket = packet.NewInitPacket(characterID, character.Room, true)

			// Add to whatever room they were in
			// TODO: Setting a character's ingest definitely doesn't work right
			// now, look into this more later
			character.Ingest = db.GetIngestID()
			pip.SAdd("room:"+character.Room+":characters", character.ID)
		} else {
			// Client provided no authentication
			return
		}

		// Add this character's id to this ingest in Redis
		db.GetInstance().SAdd("ingest:"+strconv.Itoa(character.Ingest)+":characters", characterID)

		// Wrap up
		pip.Exec()

		// Make sure SSO token is omitted from join packet that is sent to clients
		res.Name = ""
		res.QuillToken = ""
		res.Token = ""

		// Send them the relevant init packet
		data, _ := initPacket.MarshalBinary()
		m.sender.send <- data

		// Send the join packet to clients and Redis
		m.sender.character = character
		res.Character = character

		h.Send(res)
	case "message":
		// TODO: Save timestamp
		// Parse message packet
		res := packet.MessagePacket{}
		json.Unmarshal(m.msg, &res)
		res.From = m.sender.character.ID

		messageID := uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("message:"+messageID, db.StructToMap(res.Message))

		ha := fnv.New32a()
		ha.Write([]byte(res.From))
		senderHash := ha.Sum32()

		ha.Reset()
		ha.Write([]byte(res.To))
		recipientHash := ha.Sum32()

		conversationKey := "conversation:" + res.From + ":" + res.To

		if recipientHash < senderHash {
			conversationKey = "conversation:" + res.To + ":" + res.From
		}

		pip.RPush(conversationKey, messageID)
		pip.Exec()

		h.Send(res)
	case "move":
		if m.sender.character == nil {
			return
		}

		// Parse move packet
		res := packet.MovePacket{}
		json.Unmarshal(m.msg, &res)

		// Update character's position in the room
		pip := db.GetInstance().Pipeline()
		pip.HSet("character:"+m.sender.character.ID, "x", res.X)
		pip.HSet("character:"+m.sender.character.ID, "y", res.Y)
		_, err := pip.Exec()

		if err != nil {
			log.Println(err)
			log.Fatal("ERROR: Failure sending move packet to Redis")
			return
		}

		// Publish move event to other ingest servers
		res.Room = m.sender.character.Room
		res.ID = m.sender.character.ID

		h.Send(res)
	case "room_add":
		// Parse room add packet
		res := packet.RoomAddPacket{}
		json.Unmarshal(m.msg, &res)

		pip := db.GetInstance().Pipeline()
		pip.SAdd("rooms", res.ID)
		pip.HSet("room:"+res.ID, db.StructToMap(models.NewRoom(res.ID, res.Background, res.Sponsor)))
		pip.Exec()

		data, _ := res.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case "song":
		// Parse song packet
		res := packet.SongPacket{}
		json.Unmarshal(m.msg, &res)

		// Make the YouTube API call
		youtubeClient, _ := youtube.New(&http.Client{
			Transport: &transport.APIKey{Key: youtubeAPIKey},
		})

		call := youtubeClient.Videos.List([]string{"snippet", "contentDetails"}).
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
			seconds, err := strconv.Atoi(duration[minIndex+1 : secIndex])

			// Error parsing duration string
			if err != nil {
				// TODO: Send error packet
				panic(err)
			}

			res.Duration = (minutes * 60) + seconds
			res.Title = video.Snippet.Title
			res.ThumbnailURL = video.Snippet.Thumbnails.Default.Url
		}

		songID := uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("song:"+songID, db.StructToMap(res.Song))
		pip.RPush("songs", songID)
		pip.Exec()

		if err != nil {
			// TODO: Send error packet
			panic(err)
		}

		h.Send(res)
	case "teleport", "teleport_home":
		// Parse teleport packet
		res := packet.TeleportPacket{}
		json.Unmarshal(m.msg, &res)

		pip := db.GetInstance().Pipeline()

		if res.Type == "teleport_home" {
			homeExists, _ := db.GetInstance().SIsMember("rooms", "home:"+m.sender.character.ID).Result()

			if !homeExists {
				room := models.NewHomeRoom(m.sender.character.ID)
				pip.HSet("room:home:"+m.sender.character.ID, db.StructToMap(room))
				pip.SAdd("rooms", "home:"+m.sender.character.ID)
			}

			res.From = m.sender.character.Room
			res.To = "home:" + m.sender.character.ID
		}

		// Update this character's room
		pip.HSet("character:"+m.sender.character.ID, map[string]interface{}{
			"room": res.To,
			"x":    0.5,
			"y":    0.5,
		})

		// Remove this character from the previous room
		pip.SRem("room:"+m.sender.character.Room+":characters", m.sender.character.ID)
		pip.Exec()

		// Send them the init packet for this room
		initPacket := packet.NewInitPacket(m.sender.character.ID, res.To, false)
		initPacketData, _ := initPacket.MarshalBinary()
		m.sender.send <- initPacketData
		m.sender.character.Room = res.To

		// Add them to their new room
		pip = db.GetInstance().Pipeline()
		characterCmd := pip.HGetAll("character:" + m.sender.character.ID)
		pip.SAdd("room:"+res.To+":characters", m.sender.character.ID)
		pip.Exec()

		characterRes, _ := characterCmd.Result()
		var character models.Character
		db.Bind(characterRes, &character)
		character.ID = m.sender.character.ID

		// Publish event to other ingest servers
		res.Character = &character
		h.Send(res)
	case "update_map":
		// Parse update packet
		res := packet.UpdateMapPacket{}
		json.Unmarshal(m.msg, &res)

		// Update this character's location
		locationID := m.sender.character.ID

		pip := db.GetInstance().Pipeline()
		pip.HSet("location:"+locationID, db.StructToMap(res.Location))
		pip.SAdd("locations", locationID)
		pip.Exec()

		// Send locations back to client
		resp := packet.NewMapPacket()
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	}
}
