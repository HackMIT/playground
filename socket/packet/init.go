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

	// The room that the client is about to join
	Room *models.Room `json:"room"`
}

func (p *InitPacket) Init(roomSlug string) *InitPacket {
	// Fetch characters from redis
	res, _ := db.GetRejsonHandler().JSONGet("room:" + roomSlug, ".")

	var room *models.Room
	json.Unmarshal(res.([]byte), &room)

	// Set data and return
	p.BasePacket = BasePacket{Type: "init"}
	p.Room = room
	return p
}

func (p InitPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p InitPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
