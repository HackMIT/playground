package packet

import (
	"encoding/json"
	"github.com/techx/playground/db/models"
)

// Sent by clients when they're adding an element
type ElementAddPacket struct {
	BasePacket

	// The room being updated
	Room string `json:"room"`

	// The ID of the element being updated
	ID string `json:"id"`

	// The new element
	Element models.Element `json:"element"`
}

func (p ElementAddPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementAddPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
