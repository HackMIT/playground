package packet

import (
	"encoding/json"
)

// Sent when a user confirms attendance for a workshop
type WorkshopPacket struct {
	BasePacket
	
	// User who attended this workshop
	User string 

	// ID of this workshop
	Name string
}

func (p *WorkshopPacket) Init(user string, name string) *WorkshopPacket {
	p.BasePacket = BasePacket{Type: "workshop"}
	p.User = user
	p.Name = name
	return p
}

func (p WorkshopPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p WorkshopPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
