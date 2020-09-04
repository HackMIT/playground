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

func (h *Hub) disconnectClient(client *Client) {
	if client.character != nil {
		pip := db.GetInstance().Pipeline()
		pip.Del("character:" + client.character.ID + ":active")
		pip.HDel("character:"+client.character.ID, "ingest")
		pip.SRem("ingest:"+strconv.Itoa(db.GetIngestID())+":characters", client.character.ID)
		teammatesCmd := pip.SMembers("character:" + client.character.ID + ":teammates")
		friendsCmd := pip.SMembers("character:" + client.character.ID + ":friends")
		pip.Exec()

		// Remove this client from the room
		room, _ := db.GetInstance().HGet("character:"+client.character.ID, "room").Result()
		db.GetInstance().SRem("room:"+room+":characters", client.character.ID)

		// Notify others that this client left
		leavePacket := packet.NewLeavePacket(client.character, room)
		h.Send(leavePacket)

		// Tell their friends that they're offline now
		teammateIDs, _ := teammatesCmd.Result()
		friendIDs, _ := friendsCmd.Result()

		res := packet.NewStatusPacket(client.character.ID, false)
		data, _ := res.MarshalBinary()

		// TODO: This will not work with multiple ingest servers
		for _, id := range teammateIDs {
			h.SendBytes("character:"+id, data)
		}

		for _, id := range friendIDs {
			h.SendBytes("character:"+id, data)
		}
	}

	delete(h.clients, client.id)
	close(client.send)
}

// Listens for messages from websocket clients
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.id] = client
		case client := <-h.unregister:
			h.disconnectClient(client)
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
				character.ID = uuid.New().String()

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

		// Make sure character ID isn't an empty string
		if character.ID == "" {
			fmt.Println("ERROR: Empty character ID on join")
			return
		}

		// Set this character's status to active
		pip.Set("character:"+character.ID+":active", "true", 0)

		// Get info for friends notification
		teammatesCmd := pip.SMembers("character:" + character.ID + ":teammates")
		friendsCmd := pip.SMembers("character:" + character.ID + ":friends")

		// Wrap up
		pip.Exec()

		// Tell their friends that they're online now
		teammateIDs, _ := teammatesCmd.Result()
		friendIDs, _ := friendsCmd.Result()

		statusRes := packet.NewStatusPacket(character.ID, true)
		statusData, _ := statusRes.MarshalBinary()

		// TODO: This will not work with multiple ingest servers
		for _, id := range teammateIDs {
			h.SendBytes("character:"+id, statusData)
		}

		for _, id := range friendIDs {
			h.SendBytes("character:"+id, statusData)
		}

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
	case "friend_request":
		// Parse friend request packet
		res := packet.FriendRequestPacket{}
		json.Unmarshal(m.msg, &res)

		if res.RecipientID == res.SenderID {
			return
		}

		res.SenderID = m.sender.character.ID

		// Check if the other person has also sent a friend request
		isExistingRequest, _ := db.GetInstance().SIsMember("character:"+m.sender.character.ID+":requests", res.RecipientID).Result()

		if isExistingRequest {
			pip := db.GetInstance().Pipeline()
			pip.SRem("character:"+m.sender.character.ID+":requests", res.RecipientID)
			pip.SAdd("character:"+m.sender.character.ID+":friends", res.RecipientID)
			pip.SAdd("character:"+res.RecipientID+":friends", m.sender.character.ID)
			pip.Exec()

			// TODO: This will not work with more than one ingest server
			firstUpdate := packet.NewFriendUpdatePacket(res.RecipientID, m.sender.character.ID)
			data, _ := firstUpdate.MarshalBinary()
			h.SendBytes("character:"+res.RecipientID, data)

			secondUpdate := packet.NewFriendUpdatePacket(m.sender.character.ID, res.RecipientID)
			data, _ = secondUpdate.MarshalBinary()
			h.SendBytes("character:"+m.sender.character.ID, data)
		} else {
			db.GetInstance().SAdd("character:"+res.RecipientID+":requests", m.sender.character.ID)

			// TODO: This will not work with more than one ingest server
			friendUpdate := packet.NewFriendUpdatePacket(res.RecipientID, m.sender.character.ID)
			data, _ := friendUpdate.MarshalBinary()
			h.SendBytes("character:"+res.RecipientID, data)
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
		pip.HSet("song:"+songID, utils.StructToMap(res.Song))
		pip.RPush("songs", songID)
		pip.Exec()

		if err != nil {
			// TODO: Send error packet
			panic(err)
		}

		h.Send(res)
	case "status":
		// Parse status packet
		res := packet.StatusPacket{}
		json.Unmarshal(m.msg, &res)
		res.ID = m.sender.character.ID
		res.Online = true

		pip := db.GetInstance().Pipeline()

		if res.Active {
			pip.Set("character:"+m.sender.character.ID+":active", "true", 0)
		} else {
			pip.Set("character:"+m.sender.character.ID+":active", "false", 0)
		}

		teammatesCmd := pip.SMembers("character:" + m.sender.character.ID + ":teammates")
		friendsCmd := pip.SMembers("character:" + m.sender.character.ID + ":friends")
		pip.Exec()

		teammateIDs, _ := teammatesCmd.Result()
		friendIDs, _ := friendsCmd.Result()

		data, _ := res.MarshalBinary()

		// TODO: This will not work with multiple ingest servers
		for _, id := range teammateIDs {
			h.SendBytes("character:"+id, data)
		}

		for _, id := range friendIDs {
			h.SendBytes("character:"+id, data)
		}
	case "teleport", "teleport_home":
		// Parse teleport packet
		res := packet.TeleportPacket{}
		json.Unmarshal(m.msg, &res)
		res.From = m.sender.character.Room

		if res.X <= 0 || res.X >= 1 {
			res.X = 0.5
		}

		if res.Y <= 0 || res.Y >= 1 {
			res.Y = 0.5
		}

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
			"x":    res.X,
			"y":    res.Y,
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
	case "queue_pop":
		res := packet.QueuePopPacket{}
		json.Unmarshal(m.msg, &res)

		pip := db.GetInstance().Pipeline()
		characterIDCmd := pip.LPop("sponsor:" + res.SponsorID + ":hackerqueue")
		subscribers := pip.SMembers("sponsor:" + res.SponsorID + ":subscribed")
		pip.Exec()

		characterIDRes, _ := characterIDCmd.Result()
		res.CharacterID = characterIDRes
		subscriberIDs, _ := subscribers.Result()
		data, _ := res.MarshalBinary()

		// TODO @ Jack: make sure SendBytes sends to characters over all ingest servers
		for _, id := range subscriberIDs {
			h.SendBytes("character:"+id, data)
		}
	case "queue_push":
		res := packet.QueuePushPacket{}
		json.Unmarshal(m.msg, &res)

		pip := db.GetInstance().Pipeline()
		pip.RPush("sponsor:"+res.SponsorID+":hackerqueue", m.sender.character.ID)
		characterCmd := pip.HGetAll("character:" + m.sender.character.ID)
		subscribers := pip.SMembers("sponsor:" + res.SponsorID + ":subscribed")
		pip.Exec()

		characterRes, _ := characterCmd.Result()
		var character models.Character
		utils.Bind(characterRes, &character)
		character.ID = m.sender.character.ID
		res.Character = &character
		subscriberIDs, _ := subscribers.Result()
		data, _ := res.MarshalBinary()

		// TODO @ Jack: make sure SendBytes sends to characters over all ingest servers
		for _, id := range subscriberIDs {
			h.SendBytes("character:"+id, data)
		}
	case "queue_remove":
		res := packet.QueueRemovePacket{}
		json.Unmarshal(m.msg, &res)
		res.CharacterID = m.sender.character.ID

		pip := db.GetInstance().Pipeline()
		pip.LRem("sponsor:"+res.SponsorID+":hackerqueue", 0, res.CharacterID)
		subscribers := pip.SMembers("sponsor:" + res.SponsorID + ":subscribed")
		pip.Exec()

		subscriberIDs, _ := subscribers.Result()
		data, _ := res.MarshalBinary()

		// TODO @ Jack: make sure SendBytes sends to characters over all ingest servers
		for _, id := range subscriberIDs {
			h.SendBytes("character:"+id, data)
		}
	case "queue_subscribe":
		res := packet.QueueSubscribePacket{}
		json.Unmarshal(m.msg, &res)

		db.GetInstance().SAdd("sponsor:"+res.SponsorID+":subscribed", m.sender.character.ID)

		resp := packet.NewQueueSubscribePacket(res.SponsorID)
		data, _ := resp.MarshalBinary()

		h.SendBytes("character:"+m.sender.character.ID, data)
	case "queue_unsubscribe":
		res := packet.QueueUnsubscribePacket{}
		json.Unmarshal(m.msg, &res)

		db.GetInstance().SRem("sponsor:"+res.SponsorID+":subscribed", m.sender.character.ID)
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
