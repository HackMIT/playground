package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a client changes rooms
type TeleportPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The charcater who is teleporting
	Character *models.Character `json:"character"`

	// The room they're moving from
	From string `json:"from"`

	// The room they're moving to
	To string `json:"to"`

	// The resulting X coordinate
	X float64 `json:"x"`

	// The resulting Y coordinate
	Y float64 `json:"y"`
}

func NewTeleportPacket(character *models.Character, from, to string) *TeleportPacket {
	p := new(TeleportPacket)
	p.BasePacket = BasePacket{Type: "teleport"}
	p.From = from
	p.To = to
	p.Character = character
	p.X = 0.5
	p.Y = 0.5
	return p
}

func (p TeleportPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p TeleportPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p TeleportPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
