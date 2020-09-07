package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're updating the room
type ElementUpdatePacket struct {
	BasePacket
	Packet

	// The room being updated
	Room string `json:"room"`

	// The ID of the element being updated
	ID string `json:"id"`

	// The new element
	Element models.Element `json:"element"`
}

func (p ElementUpdatePacket) PermissionCheck(characterID string, role models.Role) bool {
	return role == models.Organizer
}

func (p ElementUpdatePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementUpdatePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
