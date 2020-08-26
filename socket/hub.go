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
	"time"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/socket/packet"
	"github.com/techx/playground/utils"

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

	if m.sender.character == nil && (res.Type != "auth" && res.Type != "join") {
		// No packets allowed until we've signed in
		return
	}

	switch res.Type {
	case "auth", "join":
		// Type auth is used when the character is just connecting to the socket, but not actually
		// joining a room. This is useful in limited circumstances, e.g. recording event attendance

		// Parse join packet
		res := packet.JoinPacket{}
		json.Unmarshal(m.msg, &res)

		character := new(models.Character)
		var initPacket *packet.InitPacket

		pip := db.GetInstance().Pipeline()

		if res.Name != "" {
			character = models.NewCharacter(res.Name)

			// Add character to database
			character.Ingest = db.GetIngestID()
			db.GetInstance().HSet("character:"+character.ID, utils.StructToMap(character))

			if res.Type == "join" {
				// Generate init packet before new character is added to room
				initPacket = packet.NewInitPacket(character.ID, character.Room, true)

				// Add to room:home at (0.5, 0.5)
				pip.SAdd("room:home:characters", character.ID)
			}
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
			characterID, err := db.GetInstance().HGet("quillToCharacter", quillData["id"].(string)).Result()

			if err != nil {
				// Never seen this character before, create a new one
				character = models.NewCharacterFromQuill(quillData)
				character.ID = characterID

				// Add character to database
				pip.HSet("character:"+character.ID, utils.StructToMap(character))
				pip.HSet("quillToCharacter", quillData["id"].(string), character.ID)
			} else {
				// This person has logged in before, fetch from Redis
				characterRes, _ := db.GetInstance().HGetAll("character:" + characterID).Result()
				utils.Bind(characterRes, &character)
				character.ID = characterID
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

			var characterID string

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

			utils.Bind(characterRes, character)
			character.ID = characterID
		} else {
			// Client provided no authentication data
			return
		}

		if res.Type == "join" {
			// Generate init packet before new character is added to room
			initPacket = packet.NewInitPacket(character.ID, character.Room, true)

			// Add to whatever room they were in
			pip.SAdd("room:"+character.Room+":characters", character.ID)
		}

		// Add this character's id to this ingest in Redis
		pip.SAdd("ingest:"+strconv.Itoa(character.Ingest)+":characters", character.ID)

		character.Ingest = db.GetIngestID()
		pip.HSet("character:"+character.ID, "ingest", db.GetIngestID())

		// Wrap up
		pip.Exec()

		// Authenticate the user on our end
		m.sender.character = character

		if res.Type == "join" {
			// Make sure SSO token is omitted from join packet that is sent to clients
			res.Name = ""
			res.QuillToken = ""
			res.Token = ""

			// Send them the relevant init packet
			data, _ := initPacket.MarshalBinary()
			m.sender.send <- data

			// Send the join packet to clients and Redis
			res.Character = character

			h.Send(res)
		}
	case "chat":
		res := packet.ChatPacket{}
		json.Unmarshal(m.msg, &res)

		// Check for non-ASCII characters
		if !utils.IsASCII(res.Message) {
			// TODO: Send error packet
			return
		}

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
		pip.HSet("element:"+res.ID, utils.StructToMap(res.Element))
		pip.RPush("room:"+res.Room+":elements", res.ID)
		pip.Exec()

		// Publish event to other clients
		h.Send(res)
	case "element_delete":
		// TODO: fix
		// res := packet.ElementDeletePacket{}
		// json.Unmarshal(m.msg, &res)
		// res.Room = m.sender.character.Room

		// pip := db.GetInstance().Pipeline()
		// pip.Del("element:" + res.ID)
		// pip.SRem("room:"+res.Room+":elements", res.ID)
		// pip.Exec()

		// // Publish event to other ingest servers
		// h.Send(res)
	case "element_update":
		res := packet.ElementUpdatePacket{}
		json.Unmarshal(m.msg, &res)
		res.Room = m.sender.character.Room

		if res.Element.Path == "tiles/blue1.svg" {
			res.Element.ChangingImagePath = true
			res.Element.ChangingPaths = "tiles/blue1.svg,tiles/blue2.svg,tiles/blue3.svg,tiles/blue4.svg,tiles/green1.svg,tiles/green2.svg,tiles/pink1.svg,tiles/pink2.svg,tiles/pink3.svg,tiles/pink4.svg,tiles/yellow1.svg"
			res.Element.ChangingInterval = 2000
		}

		if res.Element.Path == "djbooth.svg" {
			res.Element.Action = int(models.OpenJukebox)
		}

		db.GetInstance().HSet("element:"+res.ID, utils.StructToMap(res.Element))

		// Publish event to other ingest servers
		h.Send(res)
	case "event":
		// Parse event packet
		res := packet.EventPacket{}
		json.Unmarshal(m.msg, &res)

		isValidEvent, err := db.GetInstance().SIsMember("events", res.ID).Result()

		if !isValidEvent || err != nil {
			return
		}

		pip := db.GetInstance().Pipeline()
		pip.SAdd("event:"+res.ID+":attendees", m.sender.character.ID)
		pip.SAdd("character:"+m.sender.character.ID+":events", res.ID)
		pip.SCard("character:" + m.sender.character.ID + ":events")
		numEventsCmd := pip.HIncrBy("character:"+m.sender.character.ID+":achievements", "events", 1)
		pip.Exec()

		// Check achievement progress and update if necessary
		numEvents, err := numEventsCmd.Result()

		if numEvents == config.GetConfig().GetInt64("achievements.num_events") && err == nil {
			resp := packet.NewAchievementNotificationPacket("events")
			data, _ := resp.MarshalBinary()
			h.SendBytes("character:"+m.sender.character.ID, data)
		}
	case "get_achievements":
		// Send achievements back to client
		resp := packet.NewAchievementsPacket(m.sender.character.ID)
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
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
			utils.Bind(messageRes, messages[i])
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
		pip.HSet("hallway:"+res.ID, utils.StructToMap(res.Hallway))
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

		db.GetInstance().HSet("hallway:"+res.ID, utils.StructToMap(res.Hallway))

		// Publish event to other ingest servers
		h.Send(res)
	case "message":
		// TODO: Save timestamp
		// Parse message packet
		res := packet.MessagePacket{}
		json.Unmarshal(m.msg, &res)
		res.From = m.sender.character.ID

		// Check for non-ASCII characters
		if !utils.IsASCII(res.Message.Text) {
			// TODO: Send error packet
			return
		}

		messageID := uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("message:"+messageID, utils.StructToMap(res.Message))

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
		pip.HSet("room:"+res.ID, utils.StructToMap(models.NewRoom(res.ID, res.Background, res.Sponsor)))
		pip.Exec()

		data, _ := res.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case "settings":
		res := packet.SettingsPacket{}
		json.Unmarshal(m.msg, &res)

		db.GetInstance().HSet("character:"+m.sender.character.ID+":settings", utils.StructToMap(res.Settings))
		h.SendBytes("character:"+m.sender.character.ID, m.msg)
	case "song":
		// Parse song packet
		res := packet.SongPacket{}
		json.Unmarshal(m.msg, &res)

		if res.Remove {
			pip := db.GetInstance().Pipeline()
			pip.Del("song:"+res.ID)
			pip.LRem("songs", 1, res.ID)
			pip.Exec()
			h.Send(res)
			return
		}

		var jukeboxTimestamp time.Time
		jukeboxQuery := "character:" + m.sender.character.ID + ":jukeboxTimestamp"
		jukeboxKeyExists, _ := db.GetInstance().Exists(jukeboxQuery).Result()
		// User has never added a song to queue
		if (jukeboxKeyExists != 1) {
			jukeboxTimestamp = time.Now()
			res.RequiresWarning = true
		} else {
			timestampString, _ := db.GetInstance().Get(jukeboxQuery).Result()
			var _ error
			jukeboxTimestamp, _ = time.Parse(time.RFC3339, timestampString)
		}

		// 15 minutes has not yet passed since user last submitted a song
		if jukeboxTimestamp.After(time.Now()) {
			errorPacket := packet.NewErrorPacket(401)
			data, _ := json.Marshal(errorPacket)
			m.sender.send <- data
			return
		}

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
			minutes, _ := strconv.Atoi(duration[2:minIndex])
			seconds, _ := strconv.Atoi(duration[minIndex+1 : secIndex])

			// Song is too long
			if minutes >= 6 {
				errorPacket := packet.NewErrorPacket(400)
				data, _ := json.Marshal(errorPacket)
				m.sender.send <- data
				return
			}

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
		res.ID = songID

		jukeboxTime := time.Now().Add(time.Minute * 15)

		pip := db.GetInstance().Pipeline()
		pip.HSet("song:"+songID, utils.StructToMap(res.Song))
		pip.RPush("songs", songID)
		pip.Set(jukeboxQuery, jukeboxTime.Format(time.RFC3339), 0)
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
		res.From = m.sender.character.Room

		pip := db.GetInstance().Pipeline()

		if res.Type == "teleport_home" {
			homeExists, _ := db.GetInstance().SIsMember("rooms", "home:"+m.sender.character.ID).Result()

			if !homeExists {
				models.CreateHomeRoom(pip, m.sender.character.ID)
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
		utils.Bind(characterRes, &character)
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
		pip.HSet("location:"+locationID, utils.StructToMap(res.Location))
		pip.SAdd("locations", locationID)
		pip.Exec()

		// Send locations back to client
		resp := packet.NewMapPacket()
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	}
}
