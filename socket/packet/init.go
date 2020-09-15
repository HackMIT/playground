package packet

import (
	"encoding/json"
	"time"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v7"
)

// Sent by server to clients upon connecting. Contains information about the
// world that they load into
type InitPacket struct {
	BasePacket
	Packet

	Character *models.Character `json:"character"`

	// The room that the client is about to join
	Room *models.Room `json:"room"`

	// A token for the client to save for future authentication
	Token string `json:"token,omitempty"`

	// All possible element names
	ElementNames []string `json:"elementNames"`

	// All room names
	RoomNames []string `json:"roomNames"`

	// All of this user's friends
	Friends []Friend `json:"friends"`

	// Settings
	Settings *models.Settings `json:"settings"`

	// True if the feedback window should open
	OpenFeedback bool `json:"openFeedback"`

	// True if the user needs to register
	FirstTime bool `json:"firstTime"`
}

func NewInitPacket(characterID, roomID string, needsToken bool) *InitPacket {
	// Fetch character and room from Redis
	pip := db.GetInstance().Pipeline()
	roomCmd := pip.HGetAll("room:" + roomID)
	roomCharactersCmd := pip.SMembers("room:" + roomID + ":characters")
	roomElementsCmd := pip.LRange("room:"+roomID+":elements", 0, -1)
	roomHallwaysCmd := pip.SMembers("room:" + roomID + ":hallways")
	characterCmd := pip.HGetAll("character:" + characterID)
	settingsCmd := pip.HGetAll("character:" + characterID + ":settings")
	teammatesCmd := pip.SMembers("character:" + characterID + ":teammates")
	friendsCmd := pip.SMembers("character:" + characterID + ":friends")
	requestsCmd := pip.SMembers("character:" + characterID + ":requests")
	projectIDCmd := pip.Get("character:" + characterID + ":project")
	pip.Exec()

	room := new(models.Room).Init()
	roomRes, _ := roomCmd.Result()
	utils.Bind(roomRes, room)
	room.ID = roomID

	character := new(models.Character)
	characterRes, _ := characterCmd.Result()
	utils.Bind(characterRes, character)
	character.ID = characterID

	// Load additional room stuff
	pip = db.GetInstance().Pipeline()

	characterIDs, _ := roomCharactersCmd.Result()
	characterCmds := make([]*redis.StringStringMapCmd, len(characterIDs))

	for i, characterID := range characterIDs {
		characterCmds[i] = pip.HGetAll("character:" + characterID)
	}

	elementIDs, _ := roomElementsCmd.Result()
	elementCmds := make([]*redis.StringStringMapCmd, len(elementIDs))

	for i, elementID := range elementIDs {
		elementCmds[i] = pip.HGetAll("element:" + elementID)
	}

	hallwayIDs, _ := roomHallwaysCmd.Result()
	hallwayCmds := make([]*redis.StringStringMapCmd, len(hallwayIDs))

	for i, hallwayID := range hallwayIDs {
		hallwayCmds[i] = pip.HGetAll("hallway:" + hallwayID)
	}

	// Get friends
	teammateIDs, _ := teammatesCmd.Result()
	friendIDs, _ := friendsCmd.Result()
	requestIDs, _ := requestsCmd.Result()

	teammateCmds := make([]*redis.StringStringMapCmd, len(teammateIDs))
	friendCmds := make([]*redis.StringStringMapCmd, len(friendIDs))
	requestCmds := make([]*redis.StringStringMapCmd, len(requestIDs))

	for i, id := range teammateIDs {
		teammateCmds[i] = pip.HGetAll("character:" + id)
	}

	for i, id := range friendIDs {
		friendCmds[i] = pip.HGetAll("character:" + id)
	}

	for i, id := range requestIDs {
		requestCmds[i] = pip.HGetAll("character:" + id)
	}

	var sponsorCmd *redis.StringStringMapCmd

	if len(room.SponsorID) > 0 {
		sponsorCmd = pip.HGetAll("sponsor:" + room.SponsorID)
	}

	pip.Exec()

	for i, characterCmd := range characterCmds {
		characterRes, _ := characterCmd.Result()
		room.Characters[characterIDs[i]] = new(models.Character)
		utils.Bind(characterRes, room.Characters[characterIDs[i]])
		room.Characters[characterIDs[i]].ID = characterIDs[i]
	}

	for i, elementCmd := range elementCmds {
		elementRes, _ := elementCmd.Result()
		room.Elements = append(room.Elements, new(models.Element))
		utils.Bind(elementRes, room.Elements[i])
		room.Elements[i].ID = elementIDs[i]
	}

	for i, hallwayCmd := range hallwayCmds {
		hallwayRes, _ := hallwayCmd.Result()
		room.Hallways[hallwayIDs[i]] = new(models.Hallway)
		utils.Bind(hallwayRes, room.Hallways[hallwayIDs[i]])
	}

	if len(room.SponsorID) > 0 {
		sponsorRes, _ := sponsorCmd.Result()
		sponsor := new(models.Sponsor)
		utils.Bind(sponsorRes, sponsor)
		sponsor.ID = room.SponsorID
		room.Sponsor = sponsor
	}

	// Set data and return
	p := new(InitPacket)
	p.BasePacket = BasePacket{Type: "init"}
	p.Character = character

	feedbackOpen := time.Unix(config.GetConfig().GetInt64("feedback_open"), 0)

	if !character.FeedbackOpened && time.Now().After(feedbackOpen) {
		p.OpenFeedback = true
		db.GetInstance().HSet("character:"+characterID, "feedbackOpened", true)
	}

	p.Room = room

	// Set friends
	i := 0
	p.Friends = make([]Friend, len(teammateIDs)+len(friendIDs)+len(requestIDs))

	for j, cmd := range teammateCmds {
		data, _ := cmd.Result()
		res := new(models.Character)
		utils.Bind(data, res)

		p.Friends[i] = Friend{
			ID:       teammateIDs[j],
			Name:     res.Name,
			School:   res.School,
			Status:   0,
			Teammate: true,
			LastSeen: time.Now(),
		}

		i++
	}

	for j, cmd := range friendCmds {
		data, _ := cmd.Result()
		res := new(models.Character)
		utils.Bind(data, res)

		p.Friends[i] = Friend{
			ID:       friendIDs[j],
			Name:     res.Name,
			School:   res.School,
			Status:   0,
			LastSeen: time.Now(),
		}

		i++
	}

	for j, cmd := range requestCmds {
		data, _ := cmd.Result()
		res := new(models.Character)
		utils.Bind(data, res)

		p.Friends[i] = Friend{
			ID:       requestIDs[j],
			Name:     res.Name,
			School:   res.School,
			Status:   0,
			Pending:  true,
			LastSeen: time.Now(),
		}

		i++
	}

	if needsToken {
		// Generate a JWT
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"id": characterID,
		})

		config := config.GetConfig()
		tokenString, _ := token.SignedString([]byte(config.GetString("jwt.secret")))
		p.Token = tokenString
	}

	// Find all of the possible paths
	// TODO: Cache these
	p.ElementNames = []string{}
	// sess := session.Must(session.NewSession())
	// svc := s3.New(sess)

	// input := &s3.ListObjectsV2Input{
	// 	Bucket: aws.String("hackmit-playground-2020"),
	// 	Prefix: aws.String("elements/"),
	// }

	// result, err := svc.ListObjectsV2(input)

	// if err != nil {
	// 	p.ElementNames = []string{}
	// } else {
	// 	elementNames := make([]string, len(result.Contents)-1)

	// 	for i, item := range result.Contents {
	// 		if i == 0 {
	// 			// First key is the elements directory
	// 			continue
	// 		}

	// 		elementNames[i-1] = (*item.Key)[9:]
	// 	}

	// 	p.ElementNames = elementNames
	// }

	// Get all room names
	p.RoomNames, _ = db.GetInstance().SMembers("rooms").Result()

	// Get settings
	p.Settings = new(models.Settings)
	settingsRes, _ := settingsCmd.Result()
	utils.Bind(settingsRes, p.Settings)

	// Get project
	projectID, err := projectIDCmd.Result()

	if err != nil && len(projectID) > 0 {
		projectRes, _ := db.GetInstance().HGetAll("project:" + projectID).Result()
		utils.Bind(projectRes, p.Character)
	}

	return p
}

func (p InitPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p InitPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
