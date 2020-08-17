package packet

import (
	"encoding/json"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/s3"
	"github.com/dgrijalva/jwt-go"
    "github.com/go-redis/redis/v7"
)

// Sent by server to clients upon connecting. Contains information about the
// world that they load into
type InitPacket struct {
	BasePacket

	Character *models.Character `json:"character"`

	// The room that the client is about to join
	Room *models.Room `json:"room"`

	// A token for the client to save for future authentication
	Token string `json:"token,omitempty"`

	// All possible element names
	ElementNames []string `json:"elementNames"`

	// All room names
	RoomNames []string `json:"roomNames"`
}

func NewInitPacket(characterID, roomSlug string, needsToken bool) *InitPacket {
	// Fetch character and room from Redis
    pip := db.GetInstance().Pipeline()
    roomCmd := pip.HGetAll("room:" + roomSlug)
    roomCharactersCmd := pip.SMembers("room:" + roomSlug + ":characters")
    roomElementsCmd := pip.SMembers("room:" + roomSlug + ":elements")
    roomHallwaysCmd := pip.SMembers("room:" + roomSlug + ":hallways")
    characterCmd := pip.HGetAll("character:" + characterID)
    pip.Exec()

    room := new(models.Room).Init()
    roomRes, _ := roomCmd.Result()
    db.Bind(roomRes, room)

    character := new(models.Character)
    characterRes, _ := characterCmd.Result()
    db.Bind(characterRes, character)

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

    pip.Exec()

    for i, characterCmd := range characterCmds {
        characterRes, _ := characterCmd.Result()
        room.Characters[characterIDs[i]] = new(models.Character)
        db.Bind(characterRes, room.Characters[characterIDs[i]])
    }

    for i, elementCmd := range elementCmds {
        elementRes, _ := elementCmd.Result()
        room.Elements[elementIDs[i]] = new(models.Element)
        db.Bind(elementRes, room.Elements[elementIDs[i]])
    }

    for i, hallwayCmd := range hallwayCmds {
        hallwayRes, _ := hallwayCmd.Result()
        room.Hallways[hallwayIDs[i]] = new(models.Hallway)
        db.Bind(hallwayRes, room.Hallways[hallwayIDs[i]])
    }

	// Set data and return
	p := new(InitPacket)
	p.BasePacket = BasePacket{Type: "init"}
	p.Character = character
	p.Room = room

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
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String("hackmit-playground-2020"),
		Prefix: aws.String("elements/"),
	}

	result, err := svc.ListObjectsV2(input)

	if err != nil {
        p.ElementNames = []string{}
	} else {
        elementNames := make([]string, len(result.Contents) - 1)

        for i, item := range result.Contents {
            if i == 0 {
                // First key is the elements directory
                continue
            }

            elementNames[i - 1] = (*item.Key)[9:]
        }

        p.ElementNames = elementNames
    }

	// Get all room names
	p.RoomNames, _ = db.GetInstance().SMembers("rooms").Result()

	return p
}

func (p InitPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p InitPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
