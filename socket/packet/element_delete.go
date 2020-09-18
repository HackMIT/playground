package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're deleting an element
type ElementDeletePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The room being updated
	Room string `json:"room"`

	// The ID of the element being deleted
	ID string `json:"id"`
}

func (p ElementDeletePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && role == models.Organizer
}

func (p ElementDeletePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementDeletePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
