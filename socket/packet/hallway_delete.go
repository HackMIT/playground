package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're deleting a hallway
type HallwayDeletePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The room being updated
	Room string `json:"room"`

	// The ID of the hallway being deleted
	ID string `json:"id"`
}

func (p HallwayDeletePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && role == models.Organizer
}

func (p HallwayDeletePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p HallwayDeletePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
