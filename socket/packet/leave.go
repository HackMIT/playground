package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a client disconnects
type LeavePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The character who is leaving
	Character *models.Character `json:"character"`

	// The room that the client is leaving
	Room string `json:"room"`
}

func NewLeavePacket(character *models.Character, room string) *LeavePacket {
	p := new(LeavePacket)
	p.BasePacket = BasePacket{Type: "leave"}
	p.Character = character
	p.Room = room
	return p
}

func (p LeavePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p LeavePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p LeavePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
