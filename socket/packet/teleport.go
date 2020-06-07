package packet

import (
	"encoding/json"
)

// Sent by ingests when a client changes rooms
type TeleportPacket struct {
	BasePacket

	// The id of the client who's moving
	Id string `json:"id"`

	// The name of the client who's teleporting
	Name string `json:"name"`

	// The room they're moving from
	From string `json:"from"`

	// The room they're moving to
	To string `json:"to"`
}

func (p *TeleportPacket) Init(id string, name string, from string, to string) *TeleportPacket {
	p.BasePacket = BasePacket{Type: "teleport"}
	p.Id = id
	p.Name = name
	p.From = from
	p.To = to
	return p
}

func (p TeleportPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p TeleportPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
