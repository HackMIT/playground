package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when they need a login code
type EmailCodePacket struct {
	BasePacket
	Packet

	// The email address to check and send the code to
	Email string `json:"email"`
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
