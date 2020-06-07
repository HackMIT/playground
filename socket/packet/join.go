package packet

import (
	"encoding/json"
)

// Sent by clients after receiving the init packet. Identifies them to the
// server, and in turn other clients
type JoinPacket struct {
	BasePacket

	// The id of the client who's joining
	Id string `json:"id"`

	// The client's username
	Name string `json:"name"`

	// The client's x position (0-1)
	X float64 `json:"x"`

	// The client's y position (0-1)
	Y float64 `json:"y"`
}

func (p JoinPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p JoinPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
