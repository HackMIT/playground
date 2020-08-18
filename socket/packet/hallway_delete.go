package packet

import (
	"encoding/json"
)

// Sent by clients when they're deleting a hallway
type HallwayDeletePacket struct {
	BasePacket

	// The room being updated
	Room string `json:"room"`

	// The ID of the hallway being deleted
	ID string `json:"id"`
}

func (p HallwayDeletePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p HallwayDeletePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
