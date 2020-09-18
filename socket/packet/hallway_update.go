package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're updating a hallway
type HallwayUpdatePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The room being updated
	Room string `json:"room"`

	// The ID of the hallway being updated
	ID string `json:"id"`

	// The updated hallway
	Hallway models.Hallway `json:"hallway"`
}

func (p HallwayUpdatePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && role == models.Organizer
}

func (p HallwayUpdatePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p HallwayUpdatePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
