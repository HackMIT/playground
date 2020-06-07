package packet

import (
	"encoding/json"
)

// Sent by ingests when a client disconnects
type LeavePacket struct {
	BasePacket

	// The id of the client who's leaving
	Id string `json:"id"`

	// The name of the client who's leaving
	Name string `json:"name"`

	// The room that the client is leaving
	Room string `json:"room"`
}

func (p *LeavePacket) Init(id string, name string, room string) *LeavePacket {
	p.BasePacket = BasePacket{Type: "leave"}
	p.Id = id
	p.Name = name
	p.Room = room
	return p
}

func (p LeavePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p LeavePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
