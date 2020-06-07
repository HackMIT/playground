package packet

import (
	"encoding/json"

	"github.com/techx/playground/db"
	"github.com/techx/playground/models"
)

// Sent by server to clients upon connecting. Contains information about the
// world that they load into
type InitPacket struct {
	BasePacket

	// Map of characters that are already in the room
	Characters map[string]*models.Character `json:"characters"`
}

func (p *InitPacket) Init(roomSlug string) *InitPacket {
	// Fetch characters from redis
	data, _ := db.GetRejsonHandler().JSONGet("room:" + roomSlug, "characters")

	var characters map[string]*models.Character
	json.Unmarshal(data.([]byte), &characters)

	// Set data and return
	p.BasePacket = BasePacket{Type: "init"}
	p.Characters = characters
	return p
}

func (p InitPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p InitPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
