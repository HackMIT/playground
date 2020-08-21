package packet

import (
	"encoding/json"
)

// Sent when a user confirms attendance for an event
type EventPacket struct {
	BasePacket
	
	// User who attended this event
	User string 

	// ID of this event
	Name string
}

func (p *EventPacket) Init(user string, name string) *EventPacket {
	p.BasePacket = BasePacket{Type: "event"}
	p.User = user
	p.Name = name
	return p
}

func (p EventPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p EventPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
