package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type GetMessagesPacket struct {
	BasePacket
	Packet

	Recipient string `json:"recipient"`
}

func (p GetMessagesPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p GetMessagesPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetMessagesPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
