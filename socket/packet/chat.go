package packet

import (
	"encoding/json"
)

type ChatPacket struct {
	BasePacket

	// The id of the client who's joining
	Id string `json:"id"`

	// The client's username
	Name string `json:"name"`

	// The client's room
	Room string `json:"room"`

	Message string `json:"mssg"`
}

func (p ChatPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ChatPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}