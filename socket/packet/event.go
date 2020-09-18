package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent when a user confirms attendance for an event
type EventPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// ID of this event
	ID string `json:"id"`
}

func (p *EventPacket) Init(id string) *EventPacket {
	p.BasePacket = BasePacket{Type: "event"}
	p.ID = id
	return p
}

func (p EventPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p EventPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p EventPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
