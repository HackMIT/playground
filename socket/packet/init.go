package packet

import (
	"encoding/json"
	"fmt"

	"github.com/techx/playground/config"
	"github.com/techx/playground/db"
	"github.com/techx/playground/models"

	"github.com/dgrijalva/jwt-go"
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
}

func NewInitPacket(characterID, roomSlug string, needsToken bool) *InitPacket {
	// Fetch character and room from Redis
	res, _ := db.GetRejsonHandler().JSONGet("room:" + roomSlug, ".")
	var room *models.Room
	fmt.Println()
	json.Unmarshal(res.([]byte), &room)

	res, _ = db.GetRejsonHandler().JSONGet("character:" + characterID, ".")
	var character *models.Character
	json.Unmarshal(res.([]byte), &character)

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

	return p
}

func (p InitPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p InitPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
