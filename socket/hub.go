package socket

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
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

type ErrorCode int

const (
	BadLogin ErrorCode = iota + 1
	HighSchoolNightClub
	MissingProjectForm
	HighSchoolSponsorQueue
	MissingSurveyResponse
	NonMitMisti
)

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients
	clients map[string]*Client

	// Inbound messages from the clients
	broadcast chan *SocketMessage

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client
}

func (h *Hub) Init() *Hub {
	h.broadcast = make(chan *SocketMessage)
	h.register = make(chan *Client)
	h.unregister = make(chan *Client)
	h.clients = map[string]*Client{}
	return h
}

func (h *Hub) disconnectClient(client *Client, complete bool) {
	if client.character != nil && complete {
		pip := db.GetInstance().Pipeline()
		pip.Del("character:" + client.character.ID + ":active")
		pip.HDel("character:"+client.character.ID, "ingest")
		pip.SRem("ingest:"+db.GetIngestID()+":characters", client.character.ID)
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
		res.TeammateIDs = teammateIDs
		res.FriendIDs = friendIDs
		h.Send(res)
	}

	delete(h.clients, client.id)

	// I'm pretty sure we want to close this but it's causing an error so I'm commenting it out for now
	// close(client.send)

	client.conn.Close()
}

// Listens for messages from websocket clients
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.id] = client
		case client := <-h.unregister:
			h.disconnectClient(client, true)
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

func (h *Hub) sendSponsorQueueUpdate(sponsorID string) {
	pip := db.GetInstance().Pipeline()
	hackerSubscribersCmd := pip.LRange("sponsor:"+sponsorID+":hackerqueue", 0, -1)
	sponsorSubscribersCmd := pip.SMembers("sponsor:" + sponsorID + ":subscribed")
	pip.Exec()

	hackerIDs, _ := hackerSubscribersCmd.Result()
	sponsorIDs, _ := sponsorSubscribersCmd.Result()
	hackerCmds := make([]*redis.StringStringMapCmd, len(hackerIDs))

	pip = db.GetInstance().Pipeline()

	for i, hackerID := range hackerIDs {
		hackerCmds[i] = pip.HGetAll("subscriber:" + hackerID)
	}

	pip.Exec()

	subscribers := make([]*models.QueueSubscriber, len(hackerIDs))

	for i, hackerCmd := range hackerCmds {
		// Populate subscribers array
		subscriberRes, _ := hackerCmd.Result()
		subscribers[i] = new(models.QueueSubscriber)
		utils.Bind(subscriberRes, subscribers[i])
		subscribers[i].ID = hackerIDs[i]

		// Send queue update to each hacker
		hackerUpdatePacket := packet.NewQueueUpdateHackerPacket(sponsorID, i+1, "")
		hackerUpdatePacket.CharacterIDs = []string{hackerIDs[i]}
		h.Send(hackerUpdatePacket)
	}

	sponsorUpdatePacket := packet.NewQueueUpdateSponsorPacket(subscribers)
	sponsorUpdatePacket.CharacterIDs = sponsorIDs
	h.Send(sponsorUpdatePacket)
}

// Processes an incoming message from Redis
func (h *Hub) ProcessRedisMessage(msg []byte) {
	p, err := packet.ParsePacket(msg)

	if err != nil {
		// TODO: Log to Sentry or something -- this should never happen
		fmt.Println(err)
		log.Println("ERROR: Received invalid packet from Redis")
		return
	}

	var res map[string]interface{}
	json.Unmarshal(msg, &res)

	switch p := p.(type) {
	case packet.MessagePacket:
		h.SendBytes("character:"+p.To, msg)

		if p.To != p.From {
			h.SendBytes("character:"+p.From, msg)
		}
	case packet.ChatPacket, packet.DancePacket, packet.ElementAddPacket, packet.ElementDeletePacket, packet.ElementUpdatePacket, packet.HallwayAddPacket, packet.HallwayUpdatePacket, packet.HallwayDeletePacket, packet.MovePacket, packet.LeavePacket, packet.WardrobeChangePacket:
		h.SendBytes(res["room"].(string), msg)
	case packet.SongPacket, packet.PlaySongPacket:
		h.SendBytes("*", msg)
	case packet.FriendUpdatePacket:
		res["recipientId"] = ""
		msg, _ = json.Marshal(res)

		h.SendBytes("character:"+p.RecipientID, msg)
	case packet.JoinPacket:
		characterID := p.Character.ID
		clientID := p.ClientID

		for id, client := range h.clients {
			if client.character != nil && client.character.ID == characterID && id != clientID {
				fmt.Println("disconnecting existing client for", characterID)
				h.disconnectClient(client, false)
			}
		}

		res["clientId"] = ""
		msg, _ = json.Marshal(res)

		h.SendBytes(p.Character.Room, msg)
	case packet.QueueUpdateHackerPacket, packet.QueueUpdateSponsorPacket:
		characterIDs := res["characterIds"].([]interface{})

		res["characterIds"] = []interface{}{}
		msg, _ = json.Marshal(res)

		for _, characterID := range characterIDs {
			h.SendBytes("character:"+characterID.(string), msg)
		}
	case packet.StatusPacket:
		res["teammateIds"] = []string{}
		res["friendIds"] = []string{}
		msg, _ = json.Marshal(res)

		for _, id := range p.TeammateIDs {
			h.SendBytes("character:"+id, msg)
		}

		for _, id := range p.FriendIDs {
			h.SendBytes("character:"+id, msg)
		}
	case packet.TeleportPacket:
		leavePacket, _ := packet.NewLeavePacket(p.Character, p.From).MarshalBinary()
		h.SendBytes(p.From, leavePacket)

		joinPacket, _ := packet.NewJoinPacket(p.Character, p.To).MarshalBinary()
		h.SendBytes(p.To, joinPacket)
	}
}

// Processes an incoming message
func (h *Hub) processMessage(m *SocketMessage) {
	p, err := packet.ParsePacket(m.msg)

	if err != nil {
		// TODO: Log to Sentry or something -- this should never happen
		fmt.Println(err)
		log.Println("ERROR: Received invalid packet from", m.sender.id, "->", string(m.msg))
		return
	}

	var characterID string
	role := models.Guest

	if m.sender.character != nil {
		characterID = m.sender.character.ID
		role = models.Role(m.sender.character.Role)
	}

	if !p.PermissionCheck(characterID, role) {
		println("no permission")
		return
	}

	logID := uuid.New().String()
	logRecord := models.NewLog(characterID, string(m.msg))
	pip := db.GetInstance().Pipeline()
	pip.HSet("log:"+logID, utils.StructToMap(logRecord))
	pip.RPush("logs", logID)
	pip.Exec()

	switch p := p.(type) {
	case packet.AddEmailPacket:
		var emailsKey string

		switch models.Role(p.Role) {
		case models.SponsorRep:
			emailsKey = "sponsor_emails"
			db.GetInstance().HSet("emailToSponsor", strings.ToLower(p.Email), p.SponsorID)
		case models.Mentor:
			emailsKey = "mentor_emails"
		case models.Organizer:
			emailsKey = "organizer_emails"
		default:
			break
		}

		if emailsKey == "" {
			return
		}

		db.GetInstance().SAdd(emailsKey, strings.TrimSpace(p.Email))
	case packet.ChatPacket:
		// Check for non-ASCII characters
		if !utils.IsASCII(p.Message) {
			// TODO: Send error packet
			return
		}

		// Publish chat event to other clients
		p.Room = m.sender.character.Room
		p.ID = m.sender.character.ID
		h.Send(p)
	case packet.DancePacket:
		// Publish dance event to other clients
		p.Room = m.sender.character.Room
		p.ID = m.sender.character.ID
		h.Send(p)
	case packet.ElementTogglePacket:
		elementRes, _ := db.GetInstance().HGetAll("element:" + p.ID).Result()
		var element models.Element
		utils.Bind(elementRes, &element)
		element.ID = p.ID

		numStates := strings.Count(element.Path, ",") + 1

		if element.State < numStates-1 {
			element.State = element.State + 1
		} else {
			element.State = 0
		}

		db.GetInstance().HSet("element:"+p.ID, "state", element.State)

		// Publish update to other ingest servers
		update := packet.NewElementUpdatePacket(m.sender.character.Room, p.ID, element)
		h.Send(update)
	case packet.ElementUpdatePacket:
		p.Room = m.sender.character.Room

		if p.Element.Path == "tiles/blue1.svg" {
			p.Element.ChangingImagePath = true
			p.Element.ChangingPaths = "tiles/blue1.svg,tiles/blue2.svg,tiles/blue3.svg,tiles/blue4.svg,tiles/green1.svg,tiles/green2.svg,tiles/pink1.svg,tiles/pink2.svg,tiles/pink3.svg,tiles/pink4.svg,tiles/yellow1.svg"
			p.Element.ChangingInterval = 2000
		}

		if p.Element.Path == "djbooth.svg" {
			p.Element.Action = int(models.OpenJukebox)
		}

		db.GetInstance().HSet("element:"+p.ID, utils.StructToMap(p.Element))

		// Publish event to other ingest servers
		h.Send(p)
	case packet.EmailCodePacket:
		p.Email = strings.ToLower(p.Email)

		isValidEmail := false
		name := "mentor"
		// Make sure this email exists in our database
		switch models.Role(p.Role) {
		case models.SponsorRep:
			isValidEmail, _ = db.GetInstance().SIsMember("sponsor_emails", p.Email).Result()
			name = "sponsor"
		case models.Mentor:
			isValidEmail, _ = db.GetInstance().SIsMember("mentor_emails", p.Email).Result()
		case models.Organizer:
			isValidEmail, _ = db.GetInstance().SIsMember("organizer_emails", p.Email).Result()
		default:
			break
		}

		if !isValidEmail {
			return
		}

		code := rand.Intn(1000000)
		db.GetInstance().SAdd("login_requests", p.Email+","+strconv.Itoa(code))

		// Send email to person trying to log in
		utils.SendConfirmationEmail(p.Email, code, name)
	case packet.EventPacket:
		// Parse event packet
		res := packet.EventPacket{}
		json.Unmarshal(m.msg, &res)

		pip := db.GetInstance().Pipeline()
		validCmd := pip.SIsMember("events", res.ID)
		eventCmd := pip.HGetAll("event:" + res.ID)
		pip.Exec()

		isValidEvent, err := validCmd.Result()

		if !isValidEvent || err != nil {
			return
		}

		eventRes, _ := eventCmd.Result()
		var event models.Event
		utils.Bind(eventRes, &event)

		pip = db.GetInstance().Pipeline()
		pip.SAdd("event:"+res.ID+":attendees", m.sender.character.ID)
		pip.SAdd("character:"+m.sender.character.ID+":events", res.ID)

		var countCmd *redis.IntCmd

		if event.Type == "workshop" {
			countCmd = pip.HIncrBy("character:"+m.sender.character.ID, "numWorkshops", 1)
		} else if event.Type == "mini_event" {
			countCmd = pip.HIncrBy("character:"+m.sender.character.ID, "numMiniEvents", 1)
		}

		pip.Exec()

		if countCmd == nil {
			return
		}

		// Check achievement progress and update if necessary
		numEvents, _ := countCmd.Result()

		if event.Type == "workshop" && numEvents == config.GetConfig().GetInt64("achievements.num_workshops") {
			db.GetInstance().HSet("character:"+m.sender.character.ID+":achievements", "workshops", true)
		} else if event.Type == "mini_event" && numEvents == config.GetConfig().GetInt64("achievements.num_mini_events") {
			db.GetInstance().HSet("character:"+m.sender.character.ID+":achievements", "miniEvents", true)
		}
	case packet.FriendRequestPacket:
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
			firstNumFriendsCmd := pip.SCard("character:" + m.sender.character.ID + ":friends")
			secondNumFriendsCmd := pip.SCard("character:" + res.RecipientID + ":friends")
			pip.Exec()

			// Track achievement progress
			firstNumFriends, _ := firstNumFriendsCmd.Result()
			secondNumFriends, _ := secondNumFriendsCmd.Result()

			pip = db.GetInstance().Pipeline()

			if firstNumFriends == config.GetConfig().GetInt64("achievements.num_friends") {
				pip.HSet("character:"+m.sender.character.ID+":achievements", "hangouts", true)
			}

			if secondNumFriends == config.GetConfig().GetInt64("achievements.num_friends") {
				pip.HSet("character:"+res.RecipientID+":achievements", "hangouts", true)
			}

			pip.Exec()

			firstUpdate := packet.NewFriendUpdatePacket(res.RecipientID, m.sender.character.ID)
			h.Send(firstUpdate)

			secondUpdate := packet.NewFriendUpdatePacket(m.sender.character.ID, res.RecipientID)
			h.Send(secondUpdate)
		} else {
			db.GetInstance().SAdd("character:"+res.RecipientID+":requests", m.sender.character.ID)

			friendUpdate := packet.NewFriendUpdatePacket(res.RecipientID, m.sender.character.ID)
			h.Send(friendUpdate)
		}
	case packet.GetAchievementsPacket:
		// Send achievements back to client
		resp := packet.NewAchievementsPacket(p.ID)
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.GetMapPacket:
		// Send locations back to client
		resp := packet.NewMapPacket()
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.GetMessagesPacket:
		sender := m.sender.character.ID

		ha := fnv.New32a()
		ha.Write([]byte(sender))
		senderHash := ha.Sum32()

		ha.Reset()
		ha.Write([]byte(p.Recipient))
		recipientHash := ha.Sum32()

		conversationKey := "conversation:" + sender + ":" + p.Recipient

		if recipientHash < senderHash {
			conversationKey = "conversation:" + p.Recipient + ":" + sender
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

		resp := packet.NewMessagesPacket(messages, p.Recipient)
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.GetSponsorPacket:
		sponsorPacket := packet.NewSponsorPacket(p.SponsorID)
		data, _ := sponsorPacket.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.HallwayAddPacket:
		p.Room = m.sender.character.Room
		p.ID = uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("hallway:"+p.ID, utils.StructToMap(p.Hallway))
		pip.SAdd("room:"+p.Room+":hallways", p.ID)
		pip.Exec()

		// Publish event to other ingest servers
		h.Send(p)
	case packet.HallwayDeletePacket:
		p.Room = m.sender.character.Room

		pip := db.GetInstance().Pipeline()
		pip.Del("hallway:" + p.ID)
		pip.SRem("room:"+p.Room+":hallways", p.ID)
		pip.Exec()

		// Publish event to other ingest servers
		h.Send(p)
	case packet.HallwayUpdatePacket:
		p.Room = m.sender.character.Room

		db.GetInstance().HSet("hallway:"+p.ID, utils.StructToMap(p.Hallway))

		// Publish event to other ingest servers
		h.Send(p)
	case packet.JoinPacket:
		// Type auth is used when the character is just connecting to the socket, but not actually
		// joining a room. This is useful in limited circumstances, e.g. recording event attendance

		character := new(models.Character)
		var initPacket *packet.InitPacket
		firstTime := false

		pip := db.GetInstance().Pipeline()

		if p.QuillToken != "" {
			// Fetch data from Quill
			quillValues := map[string]string{
				"token": p.QuillToken,
			}

			quillBody, _ := json.Marshal(quillValues)
			// TODO: Error handling
			resp, _ := http.Post("https://my.hackmit.org/auth/sso/exchange", "application/json", bytes.NewBuffer(quillBody))

			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)

			// var quillData map[string]interface{}
			var quillData models.QuillResponse
			err := json.Unmarshal(body, &quillData)

			if err != nil {
				// Likely invalid SSO token
				// TODO: Send error packet
				return
			}

			if !quillData.Status.Admitted || !quillData.Status.Confirmed {
				// Don't allow non-admitted hackers to access Playground
				// TODO: Send error packet
				return
			}

			// Load this client's character
			characterID, err := db.GetInstance().HGet("quillToCharacter", quillData.ID).Result()

			if err != nil {
				// Never seen this character before, create a new one
				character = models.NewCharacterFromQuill(quillData.Profile)
				character.Email = quillData.Email
				character.ID = uuid.New().String()

				// Add character to database
				pip.HSet("character:"+character.ID, utils.StructToMap(character))
				pip.HSet("quillToCharacter", quillData.ID, character.ID)
				pip.HSet("emailToCharacter", quillData.Email, character.ID)

				// Make sure they get the account setup screen
				firstTime = true
			} else {
				// This person has logged in before, fetch from Redis
				characterRes, _ := db.GetInstance().HGetAll("character:" + characterID).Result()
				utils.Bind(characterRes, character)
				character.ID = characterID
			}
		} else if p.Token != "" {
			// TODO: Error handling
			token, err := jwt.Parse(p.Token, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}

				return []byte(config.GetSecret("JWT_SECRET")), nil
			})

			if err != nil {
				errorPacket := packet.NewErrorPacket(int(BadLogin))
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
				errorPacket := packet.NewErrorPacket(int(BadLogin))
				data, _ := json.Marshal(errorPacket)
				m.sender.send <- data
				return
			}

			utils.Bind(characterRes, character)
			character.ID = characterID
		} else if p.Email != "" {
			isValidLoginRequest, _ := db.GetInstance().SIsMember("login_requests", p.Email+","+strconv.Itoa(p.Code)).Result()

			if !isValidLoginRequest {
				return
			}

			// Load this client's character
			characterID, err := db.GetInstance().HGet("emailToCharacter", p.Email).Result()

			if err != nil {
				// Never seen this character before, create a new one
				character = models.NewCharacter("Player")
				character.Email = p.Email
				character.ID = uuid.New().String()

				// Check this character's role
				rolePip := db.GetInstance().Pipeline()
				sponsorCmd := rolePip.SIsMember("sponsor_emails", p.Email)
				sponsorIDCmd := rolePip.HGet("emailToSponsor", p.Email)
				mentorCmd := rolePip.SIsMember("mentor_emails", p.Email)
				organizerCmd := rolePip.SIsMember("organizer_emails", p.Email)
				rolePip.Exec()

				isSponsor, _ := sponsorCmd.Result()
				isMentor, _ := mentorCmd.Result()
				isOrganizer, _ := organizerCmd.Result()

				if isSponsor {
					character.Role = int(models.SponsorRep)

					sponsorID, _ := sponsorIDCmd.Result()
					character.SponsorID = sponsorID
				} else if isMentor {
					character.Role = int(models.Mentor)
				} else if isOrganizer {
					character.Role = int(models.Organizer)
				}

				// Add character to database
				pip.HSet("character:"+character.ID, utils.StructToMap(character))
				pip.HSet("emailToCharacter", p.Email, character.ID)

				// Make sure they get the account setup screen
				firstTime = true
			} else {
				// This person has logged in before, fetch from Redis
				characterRes, _ := db.GetInstance().HGetAll("character:" + characterID).Result()
				utils.Bind(characterRes, &character)
				character.ID = characterID
			}

			p.Email = ""
			p.Code = 0
		} else {
			// Client provided no authentication data
			return
		}

		if p.Type == "join" {
			pip.Exec()

			// Generate init packet before new character is added to room
			initPacket = packet.NewInitPacket(character.ID, character.Room, true)
			initPacket.FirstTime = firstTime

			// Add to whatever room they were in
			pip.SAdd("room:"+character.Room+":characters", character.ID)
		}

		// Add this character's id to this ingest in Redis
		pip.SAdd("ingest:"+character.Ingest+":characters", character.ID)

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
		statusRes := packet.NewStatusPacket(character.ID, true)
		statusRes.FriendIDs, _ = friendsCmd.Result()
		statusRes.TeammateIDs, _ = teammatesCmd.Result()
		h.Send(statusRes)

		// Authenticate the user on our end
		m.sender.character = character

		if p.Type == "join" {
			// Make sure SSO token is omitted from join packet that is sent to clients
			p.Name = ""
			p.QuillToken = ""
			p.Token = ""

			// Send them the relevant init packet
			data, _ := initPacket.MarshalBinary()
			m.sender.send <- data

			// Send the join packet to clients and Redis
			p.Character = character
			p.ClientID = m.sender.id
			p.Room = character.Room

			if strings.HasPrefix(p.Room, "arena:") {
				p.SetProject()
			}

			h.Send(p)
		}
	case packet.GetCurrentSongPacket:
		queueRes, _ := db.GetInstance().Get("queuestatus").Result()
		queueStatusInt, _ := strconv.Atoi(queueRes)
		queueStatus := int64(queueStatusInt)
		currentSongID, _ := db.GetInstance().Get("currentsong").Result()

		songRes, _ := db.GetInstance().HGetAll("song:" + currentSongID).Result()
		var currentSong models.Song
		utils.Bind(songRes, &currentSong)

		songEnd := time.Unix(queueStatus, 0)
		timeDiff := songEnd.Sub(time.Now())
		var songStart int
		if currentSong.Duration != 0 {
			songStart = currentSong.Duration - int(timeDiff.Seconds())
		} else {
			songStart = 0
		}
		currentSong.ID = currentSongID

		resp := packet.NewPlaySongPacket(&currentSong, songStart)
		h.Send(resp)
	case packet.GetSongsPacket:
		songIDs, _ := db.GetInstance().LRange("songs", 0, -1).Result()

		pip := db.GetInstance().Pipeline()
		songCmds := make([]*redis.StringStringMapCmd, len(songIDs))

		for i, songID := range songIDs {
			songCmds[i] = pip.HGetAll("song:" + songID)
		}

		pip.Exec()
		songs := make([]*models.Song, len(songIDs))

		for i, songCmd := range songCmds {
			songRes, _ := songCmd.Result()
			songs[i] = new(models.Song)
			utils.Bind(songRes, songs[i])
			songs[i].ID = songIDs[i]
		}

		resp := packet.NewSongsPacket(songs)
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.MessagePacket:
		// TODO: Save timestamp
		p.From = m.sender.character.ID

		// Check for non-ASCII characters
		if !utils.IsASCII(p.Message.Text) {
			// TODO: Send error packet
			return
		}

		messageID := uuid.New().String()

		pip := db.GetInstance().Pipeline()
		pip.HSet("message:"+messageID, utils.StructToMap(p.Message))

		ha := fnv.New32a()
		ha.Write([]byte(p.From))
		senderHash := ha.Sum32()

		ha.Reset()
		ha.Write([]byte(p.To))
		recipientHash := ha.Sum32()

		conversationKey := "conversation:" + p.From + ":" + p.To

		if recipientHash < senderHash {
			conversationKey = "conversation:" + p.To + ":" + p.From
		}

		pip.RPush(conversationKey, messageID)
		pip.Exec()

		h.Send(p)
	case packet.MovePacket:
		if m.sender.character == nil {
			return
		}

		// Update character's position in the room
		pip := db.GetInstance().Pipeline()
		pip.HSet("character:"+m.sender.character.ID, "x", p.X)
		pip.HSet("character:"+m.sender.character.ID, "y", p.Y)
		_, err := pip.Exec()

		if err != nil {
			log.Println(err)
			log.Fatal("ERROR: Failure sending move packet to Redis")
			return
		}

		// Publish move event to other ingest servers
		p.Room = m.sender.character.Room
		p.ID = m.sender.character.ID

		h.Send(p)
	case packet.ProjectFormPacket:
		projectID := uuid.New().String()

		pip := db.GetInstance().Pipeline()

		characterIDCmds := make([]*redis.StringCmd, len(p.Teammates))

		for i, email := range p.Teammates {
			characterIDCmds[i] = pip.HGet("emailToCharacter", email)
		}

		pip.Exec()
		pip = db.GetInstance().Pipeline()

		p.Project.Challenges = strings.Join(p.Challenges, ",")
		p.Project.Emails = strings.Join(append(p.Teammates, m.sender.character.Email), ",")
		p.Project.SubmittedAt = int(time.Now().Unix())
		pip.HSet("project:"+projectID, utils.StructToMap(p.Project))

		for _, cmd := range characterIDCmds {
			characterID, err := cmd.Result()

			if err != nil || len(characterID) == 0 {
				continue
			}

			pip.Set("character:"+characterID+":project", projectID, 0)
			pip.HSet("character:"+characterID+":achievements", "trackCounter", true)
		}

		pip.Set("character:"+m.sender.character.ID+":project", projectID, 0)
		pip.HSet("character:"+m.sender.character.ID+":achievements", "trackCounter", true)
		pip.Exec()
	case packet.RegisterPacket:
		pip := db.GetInstance().Pipeline()

		if p.Name != "" {
			if m.sender.character.Role == int(models.Organizer) {
				p.Name += " (Blueprint)"
			} else if m.sender.character.Role == int(models.SponsorRep) {
				sponsorName, _ := db.GetInstance().HGet("sponsor:"+m.sender.character.SponsorID, "name").Result()
				p.Name += " (" + sponsorName + ")"
			}

			pip.HSet("character:"+m.sender.character.ID, "name", p.Name)
		}

		if p.Location != "" {
			pip.HSet("character:"+m.sender.character.ID, "location", p.Location)
		}

		if p.Bio != "" {
			pip.HSet("character:"+m.sender.character.ID, "bio", p.Bio)
		}

		if p.PhoneNumber != "" {
			pip.HSet("character:"+m.sender.character.ID+":settings", "phoneNumber", p.PhoneNumber)
		}

		roomCmd := pip.HGet("character:"+m.sender.character.ID, "room")
		pip.Exec()
		room, _ := roomCmd.Result()

		initPacket := packet.NewInitPacket(m.sender.character.ID, room, true)
		data, _ := initPacket.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.SettingsPacket:
		pip := db.GetInstance().Pipeline()

		if len(p.Settings.TwitterHandle) > 0 && p.CheckTwitter {
			url := "https://api.twitter.com/2/tweets/search/recent?query=from:" + p.Settings.TwitterHandle + "&tweet.fields=entities"
			method := "GET"

			bearer := "Bearer " + config.GetSecret("TWITTER_API_KEY")
			client := &http.Client{}
			req, err := http.NewRequest(method, url, nil)

			req.Header.Add("Authorization", bearer)
			if err != nil {
				fmt.Println("errror", err)
			}
			res, err := client.Do(req)
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)

			// Track achievements
			usedHashtag := strings.Contains(strings.ToLower(string(body)), "#hackmit2020")
			usedMemeHashtag := strings.Contains(strings.ToLower(string(body)), "#hackmitmemes")

			if usedHashtag {
				pip.HSet("character:"+m.sender.character.ID+":achievements", "socialMedia", true)
			}

			if usedMemeHashtag {
				pip.HSet("character:"+m.sender.character.ID+":achievements", "memeLord", true)
			}
		}

		if p.Location != "" {
			pip.HSet("character:"+m.sender.character.ID, "location", p.Location)
		}

		if p.Bio != "" {
			pip.HSet("character:"+m.sender.character.ID, "bio", p.Bio)
		}

		if p.Zoom != "" {
			pip.HSet("character:"+m.sender.character.ID, "zoom", p.Zoom)
		}

		pip.HSet("character:"+m.sender.character.ID+":settings", utils.StructToMap(p.Settings))
		pip.Exec()

		h.SendBytes("character:"+m.sender.character.ID, m.msg)
	case packet.SongPacket:
		// Parse song packet
		if p.Remove {
			pip := db.GetInstance().Pipeline()
			pip.Del("song:" + p.ID)
			pip.LRem("songs", 1, p.ID)
			pip.Exec()
			h.Send(p)
			return
		}

		var jukeboxTimestamp time.Time
		jukeboxQuery := "character:" + m.sender.character.ID + ":jukeboxTimestamp"
		jukeboxKeyExists, _ := db.GetInstance().Exists(jukeboxQuery).Result()
		if jukeboxKeyExists != 1 {
			// User has never added a song to queue -- remind them of COC
			jukeboxTimestamp = time.Now()
			warningPacket := packet.NewJukeboxWarningPacket()
			data, _ := json.Marshal(warningPacket)
			m.sender.send <- data
		} else {
			// User has added a song to the queue before -- no need for a warning
			timestampString, _ := db.GetInstance().Get(jukeboxQuery).Result()
			jukeboxTimestamp, _ = time.Parse(time.RFC3339, timestampString)
		}

		// 15 minutes has not yet passed since user last submitted a song
		if m.sender.character.Role != int(models.Organizer) && jukeboxTimestamp.After(time.Now()) {
			errorPacket := packet.NewErrorPacket(401)
			data, _ := json.Marshal(errorPacket)
			m.sender.send <- data
			return
		}

		// Make the YouTube API call
		youtubeClient, _ := youtube.New(&http.Client{
			Transport: &transport.APIKey{Key: config.GetSecret(config.YouTubeKey)},
		})

		call := youtubeClient.Videos.List([]string{"snippet", "contentDetails"}).
			Id(p.VidCode)

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
			var minutes int
			var seconds int
			var _ error
			if minIndex != -1 {
				minutes, _ = strconv.Atoi(duration[2:minIndex])
				seconds, _ = strconv.Atoi(duration[minIndex+1 : secIndex])
			} else {
				minutes = 0
				timeIndex := strings.Index(duration, "T")
				seconds, _ = strconv.Atoi(duration[timeIndex+1 : secIndex])
			}

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

			p.Duration = (minutes * 60) + seconds
			p.Title = video.Snippet.Title
			p.ThumbnailURL = video.Snippet.Thumbnails.Default.Url
		}

		songID := uuid.New().String()
		p.ID = songID

		jukeboxTime := time.Now().Add(time.Minute * 15)

		pip := db.GetInstance().Pipeline()
		pip.HSet("song:"+songID, utils.StructToMap(p.Song))
		pip.RPush("songs", songID)
		pip.Set(jukeboxQuery, jukeboxTime.Format(time.RFC3339), 0)
		pip.Exec()

		if err != nil {
			// TODO: Send error packet
			panic(err)
		}

		h.Send(p)
	case packet.StatusPacket:
		if m.sender.character == nil {
			return
		}

		p.ID = m.sender.character.ID
		p.Online = true

		pip := db.GetInstance().Pipeline()

		if p.Active {
			pip.Set("character:"+m.sender.character.ID+":active", "true", 0)
		} else {
			pip.Set("character:"+m.sender.character.ID+":active", "false", 0)
		}

		teammatesCmd := pip.SMembers("character:" + m.sender.character.ID + ":teammates")
		friendsCmd := pip.SMembers("character:" + m.sender.character.ID + ":friends")
		pip.Exec()

		p.FriendIDs, _ = friendsCmd.Result()
		p.TeammateIDs, _ = teammatesCmd.Result()
		h.Send(p)
	case packet.TeleportPacket:
		p.From = m.sender.character.Room

		if p.X <= 0 || p.X >= 1 {
			p.X = 0.5
		}

		if p.Y <= 0 || p.Y >= 1 {
			p.Y = 0.5
		}

		pip := db.GetInstance().Pipeline()

		if p.Type == "teleport_home" {
			p.From = m.sender.character.Room

			if m.sender.character.SponsorID != "" {
				// If this character is a sponsor rep, send them to their sponsor room
				p.To = "sponsor:" + m.sender.character.SponsorID
			} else {
				// Otherwise, send them to their personal room
				homeExists, _ := db.GetInstance().SIsMember("rooms", "home:"+m.sender.character.ID).Result()

				if !homeExists {
					db.CreateRoom("home:"+m.sender.character.ID, db.Personal)
				}

				p.To = "home:" + m.sender.character.ID
			}
		}

		if strings.HasPrefix(p.To, "character:") {
			characterID := strings.Split(p.To, ":")[1]

			pip := db.GetInstance().Pipeline()
			isFriendCmd := pip.SIsMember("character:"+m.sender.character.ID+":friends", characterID)
			roomCmd := pip.HGet("character:"+characterID, "room")
			pip.Exec()

			isFriend, _ := isFriendCmd.Result()

			if !isFriend {
				// Don't let people teleport to random other people
				return
			}

			p.To, _ = roomCmd.Result()
		}

		if p.To == "nightclub" && (!m.sender.character.IsCollege && m.sender.character.Role != int(models.Organizer)) {
			errorPacket := packet.NewErrorPacket(int(HighSchoolNightClub))
			data, _ := json.Marshal(errorPacket)
			m.sender.send <- data
			return
		}

		if p.To == "misti" && m.sender.character.School != "Massachusetts Institute of Technology" && m.sender.character.Role != int(models.Organizer) {
			errorPacket := packet.NewErrorPacket(int(NonMitMisti))
			data, _ := json.Marshal(errorPacket)
			m.sender.send <- data
			return
		}

		var project *models.Project

		// If we're going to the hacker arena after 5pm, add the character's project
		if strings.HasPrefix(p.To, "arena:") && time.Now().Unix() >= 1600549200 {
			projectID, _ := db.GetInstance().Get("character:" + m.sender.character.ID + ":project").Result()

			if m.sender.character.Role == int(models.Hacker) && len(projectID) == 0 {
				errorPacket := packet.NewErrorPacket(int(MissingSurveyResponse))
				data, _ := json.Marshal(errorPacket)
				m.sender.send <- data
				return
			}

			pip := db.GetInstance().Pipeline()

			// Make sure they earn the peer expo achievement
			pip.HSet("character:"+m.sender.character.ID+":achievements", "peerExpo", true)

			projectCmd := pip.HGetAll("project:" + projectID)
			pip.Exec()

			projectRes, _ := projectCmd.Result()
			project = new(models.Project)
			utils.Bind(projectRes, project)
		}

		// Update this character's room
		pip.HSet("character:"+m.sender.character.ID, map[string]interface{}{
			"room": p.To,
			"x":    p.X,
			"y":    p.Y,
		})

		// Remove this character from the previous room
		pip.SRem("room:"+m.sender.character.Room+":characters", m.sender.character.ID)
		pip.Exec()

		// Send them the init packet for this room
		initPacket := packet.NewInitPacket(m.sender.character.ID, p.To, false)
		initPacketData, _ := initPacket.MarshalBinary()
		m.sender.send <- initPacketData
		m.sender.character.Room = p.To

		// Add them to their new room
		pip = db.GetInstance().Pipeline()
		characterCmd := pip.HGetAll("character:" + m.sender.character.ID)
		pip.SAdd("room:"+p.To+":characters", m.sender.character.ID)
		pip.Exec()

		characterRes, _ := characterCmd.Result()
		var character models.Character
		utils.Bind(characterRes, &character)
		character.ID = m.sender.character.ID

		// Publish event to other ingest servers
		p.Character = &character
		p.Character.Project = project

		// If we're entering a sponsor room, track achievement progress
		if strings.HasPrefix(p.To, "sponsor:") {
			numSponsors, _ := db.GetInstance().HIncrBy("character:"+m.sender.character.ID, "numSponsorsVisited", 1).Result()

			if numSponsors == config.GetConfig().GetInt64("achievements.num_sponsors") {
				db.GetInstance().HSet("character:"+m.sender.character.ID+":achievements", "companyTour", true)
			}
		}

		h.Send(p)
	case packet.QueueJoinPacket:
		sponsorRes, _ := db.GetInstance().HGetAll("sponsor:" + p.SponsorID).Result()
		var sponsor models.Sponsor
		utils.Bind(sponsorRes, &sponsor)

		if !sponsor.QueueOpen {
			return
		}

		hackerIDs, _ := db.GetInstance().LRange("sponsor:"+p.SponsorID+":hackerqueue", 0, -1).Result()

		for _, hackerID := range hackerIDs {
			if hackerID == m.sender.character.ID {
				// This hacker is already in the queue
				return
			}
		}

		if !m.sender.character.IsCollege && m.sender.character.Role != int(models.Organizer) {
			errorPacket := packet.NewErrorPacket(int(HighSchoolSponsorQueue))
			data, _ := json.Marshal(errorPacket)
			m.sender.send <- data
			return
		}

		pip := db.GetInstance().Pipeline()
		pip.RPush("sponsor:"+p.SponsorID+":hackerqueue", m.sender.character.ID)
		pip.HSet("character:"+m.sender.character.ID, "queueId", p.SponsorID)

		subscriber := models.NewQueueSubscriber(m.sender.character, p.Interests)
		pip.HSet("subscriber:"+m.sender.character.ID, utils.StructToMap(subscriber))

		// Track achievements
		pip.HSet("character:"+m.sender.character.ID+":achievements", "sponsorQueue", true)
		pip.Exec()

		h.sendSponsorQueueUpdate(p.SponsorID)
	case packet.QueueRemovePacket:
		pip := db.GetInstance().Pipeline()
		pip.LRem("sponsor:"+p.SponsorID+":hackerqueue", 0, p.CharacterID)
		pip.HSet("character:"+p.CharacterID, "queueId", "")
		phoneCmd := pip.HGet("character:"+p.CharacterID+":settings", "phoneNumber")
		sponsorNameCmd := pip.HGet("sponsor:"+p.SponsorID, "name")
		pip.Exec()

		h.sendSponsorQueueUpdate(p.SponsorID)

		if m.sender.character.Role == int(models.SponsorRep) {
			// If a sponsor took a hacker off the queue, send them the sponsor's URL
			// TODO: Replace this with the sponsor's actual URL
			hackerUpdatePacket := packet.NewQueueUpdateHackerPacket(p.SponsorID, 0, p.Zoom)
			hackerUpdatePacket.CharacterIDs = []string{p.CharacterID}
			h.Send(hackerUpdatePacket)

			// Send the hacker a text message letting them know it's their turn
			phoneNumber, _ := phoneCmd.Result()
			sponsorName, _ := sponsorNameCmd.Result()

			if len(phoneNumber) == 0 || len(sponsorName) == 0 {
				return
			}

			reg, _ := regexp.Compile("[^0-9]+")
			phoneNumber = "+1" + reg.ReplaceAllString(phoneNumber, "")

			msgData := url.Values{}
			msgData.Set("To", phoneNumber)
			msgData.Set("From", config.GetConfig().GetString("twilio.from_phone_number"))
			msgData.Set("Body", "It's your turn to talk to "+sponsorName+"! Meet with them at "+p.Zoom)
			msgDataReader := *strings.NewReader(msgData.Encode())

			client := &http.Client{}
			urlStr := "https://api.twilio.com/2010-04-01/Accounts/" + config.GetSecret(config.TwilioAccountSID) + "/Messages.json"
			req, _ := http.NewRequest("POST", urlStr, &msgDataReader)
			req.SetBasicAuth(config.GetSecret(config.TwilioAccountSID), config.GetSecret(config.TwilioAuthToken))
			req.Header.Add("Accept", "application/json")
			req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			client.Do(req)
		}
	case packet.QueueSubscribePacket:
		db.GetInstance().SAdd("sponsor:"+p.SponsorID+":subscribed", m.sender.character.ID)

		// TODO: This is inefficient, we should just send the update to the newly subscribed sponsor
		h.sendSponsorQueueUpdate(p.SponsorID)
	case packet.QueueUnsubscribePacket:
		db.GetInstance().SRem("sponsor:"+p.SponsorID+":subscribed", m.sender.character.ID)
	case packet.ReportPacket:
		json := []byte(`{"text": "` + m.sender.character.Name + `: ` + `(` + p.CharacterID + `): ` + p.Text + `"}`)
		body := bytes.NewBuffer(json)

		client := &http.Client{}
		req, _ := http.NewRequest("POST", config.GetSecret("SLACK_WEBHOOK"), body)
		req.Header.Add("Content-Type", "application/json; charset=utf-8")
		client.Do(req)
	case packet.UpdateMapPacket:
		// Update this character's location
		locationID := m.sender.character.ID

		pip := db.GetInstance().Pipeline()
		pip.HSet("location:"+locationID, utils.StructToMap(p.Location))
		pip.SAdd("locations", locationID)

		// Track achievements
		pip.HSet("character:"+m.sender.character.ID+":achievements", "sendLocation", true)
		pip.Exec()

		// Send locations back to client
		resp := packet.NewMapPacket()
		data, _ := resp.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.UpdateSponsorPacket:
		pip := db.GetInstance().Pipeline()

		if len(p.Sponsor.Challenges) > 0 {
			pip.HSet("sponsor:"+m.sender.character.SponsorID, "challenges", p.Sponsor.Challenges)
		}

		if len(p.Sponsor.Description) > 0 {
			pip.HSet("sponsor:"+m.sender.character.SponsorID, "description", p.Sponsor.Description)
		}

		if len(p.Sponsor.URL) > 0 {
			pip.HSet("sponsor:"+m.sender.character.SponsorID, "url", p.Sponsor.URL)
		}

		if p.SetQueueOpen {
			pip.HSet("sponsor:"+m.sender.character.SponsorID, "queueOpen", p.QueueOpen)
		}

		pip.Exec()

		// Send new sponsor packet
		sponsorPacket := packet.NewSponsorPacket(m.sender.character.SponsorID)
		data, _ := sponsorPacket.MarshalBinary()
		h.SendBytes("character:"+m.sender.character.ID, data)
	case packet.WardrobeChangePacket:
		p.CharacterID = m.sender.character.ID
		p.Room = m.sender.character.Room

		db.GetInstance().HSet("character:"+m.sender.character.ID, map[string]interface{}{
			"eyeColor":   p.EyeColor,
			"skinColor":  p.SkinColor,
			"shirtColor": p.ShirtColor,
			"pantsColor": p.PantsColor,
		})

		h.Send(p)
	}
}
