package packet

import (
	"encoding/json"
	"github.com/techx/playground/db/models"
)

// Sent by clients when adding a hallway
type HallwayAddPacket struct {
	BasePacket

	// The room being updated
	Room string `json:"room"`

	// The ID of the hallway being updated
	ID string `json:"id"`

	// The new hallway
	Hallway models.Hallway `json:"hallway"`
}

func (p HallwayAddPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p HallwayAddPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
