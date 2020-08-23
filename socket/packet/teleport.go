package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a client changes rooms
type TeleportPacket struct {
	BasePacket

	// The charcater who is teleporting
	Character *models.Character `json:"character"`

	// The room they're moving from
	From string `json:"from"`

	// The room they're moving to
	To string `json:"to"`
}

func NewTeleportPacket(character *models.Character, from, to string) *TeleportPacket {
	p := new(TeleportPacket)
	p.BasePacket = BasePacket{Type: "teleport"}
	p.From = from
	p.To = to
	p.Character = character
	return p
}

func (p TeleportPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p TeleportPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
