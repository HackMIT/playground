package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they're deleting an element
type ElementTogglePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The ID of the element being toggled
	ID string `json:"id"`
}

func (p ElementTogglePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p ElementTogglePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementTogglePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
