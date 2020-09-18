package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're adding an element
type ElementAddPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The room being updated
	Room string `json:"room"`

	// The ID of the element being updated
	ID string `json:"id"`

	// The new element
	Element models.Element `json:"element"`
}

func (p ElementAddPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && role == models.Organizer
}

func (p ElementAddPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementAddPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
