package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they need a login code
type EmailCodePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	// The email address to check and send the code to
	Email string `json:"email"`

	// The role this user is signing up for (see models.Role)
	Role int `json:"role"`
}

func (p EmailCodePacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p EmailCodePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p EmailCodePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
