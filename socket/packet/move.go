package packet

import (
	"encoding/json"
)

// Sent by clients when they move around
type MovePacket struct {
	BasePacket

	// The id of the  client who is moving
	Id string `json:"id"`

	// The client's x position (0-1)
	X float64 `json:"x"`

	// The client's y position (0-1)
	Y float64 `json:"y"`
}

func (p MovePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MovePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
