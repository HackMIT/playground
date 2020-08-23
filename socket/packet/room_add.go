package packet

import (
	"encoding/json"
)

// Sent by clients when they're adding an element
type RoomAddPacket struct {
	BasePacket

	// The name of the new room
	ID string `json:"id"`

	// The background path for this room
	Background string `json:"background"`

	// True if this is a sponsor room
	Sponsor bool `json:"sponsor"`
}

func (p RoomAddPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p RoomAddPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
