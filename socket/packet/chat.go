package packet

import (
	"encoding/json"
)

type ChatPacket struct {
	BasePacket

	// The message being sent
	Message string `json:"mssg"`

	// The id of the client who's joining
	ID string `json:"id"`

	// The client's room
	Room string `json:"room"`
}

func (p ChatPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ChatPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
