package packet

import (
	"encoding/json"
)

// Sent by clients when they move around
type HackerqueuePacket struct {
	BasePacket

	// The id of the client who is joining
	ID string `json:"id"`

	// The room that the client is in
	Room string `json:"room"`
}

func (p HackerqueuePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p HackerqueuePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
