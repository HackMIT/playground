package packet

import (
	"encoding/json"
)

// Sent by ingests when a client changes rooms
type ChangeRoomPacket struct {
	BasePacket

	// The id of the client who's moving
	Id string `json:"id"`

	// The room they're moving from
	From string `json:"from"`

	// The room they're moving to
	To string `json:"to"`
}

func (p *ChangeRoomPacket) Init(id string, from string, to string) *ChangeRoomPacket {
	p.BasePacket = BasePacket{Type: "change"}
	p.Id = id
	p.From = from
	p.To = to
	return p
}

func (p ChangeRoomPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ChangeRoomPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
