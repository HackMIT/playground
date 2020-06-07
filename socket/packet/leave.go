package packet

import (
	"encoding/json"
)

// Sent by ingests when a client disconnects
type LeavePacket struct {
	BasePacket

	// The id of the client who's leaving
	Id string `json:"id"`
}

func (p *LeavePacket) Init(id string) *LeavePacket {
	p.BasePacket = BasePacket{Type: "leave"}
	p.Id = id
	return p
}

func (p LeavePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p LeavePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
