package packet

import (
	"encoding/json"
)

// Sent by clients when they're deleting an element
type ElementDeletePacket struct {
	BasePacket

	// The room being updated
	Room string `json:"room"`

	// The ID of the element being deleted
	ID string `json:"id"`
}

func (p ElementDeletePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ElementDeletePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
