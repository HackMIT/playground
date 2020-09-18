package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're adding an element
type RoomAddPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The name of the new room
	ID string `json:"id"`

	// The background path for this room
	Background string `json:"background"`

	// True if this is a sponsor room
	Sponsor bool `json:"sponsor"`
}

func (p RoomAddPacket) PermissionCheck(characterID string, role models.Role) bool {
	return role == models.Organizer
}

func (p RoomAddPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p RoomAddPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
