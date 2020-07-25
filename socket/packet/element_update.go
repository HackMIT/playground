package packet

import (
	"encoding/json"
	"github.com/techx/playground/models"
)

// Sent by clients when they're updating the room
type ElementUpdatePacket struct {
	BasePacket

	// The slug of the room being updated
	Slug string `json:"slug"`

	// The ID of the element being updated
	ID string `json:"id"`

	// The new element
	Element models.Element `json:"element"`
}

func (p ElementUpdatePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementUpdatePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
