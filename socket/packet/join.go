package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients after receiving the init packet. Identifies them to the
// server, and in turn other clients
type JoinPacket struct {
	BasePacket

	// Client attributes
	Name       string `json:"name,omitempty"`
	QuillToken string `json:"quillToken,omitempty"`
	Token      string `json:"token,omitempty"`

	// Server attributes
	Character *models.Character `json:"character"`
}

func NewJoinPacket(character *models.Character) *JoinPacket {
	p := new(JoinPacket)
	p.BasePacket = BasePacket{Type: "join"}
	p.Character = character
	return p
}

func (p JoinPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p JoinPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
