package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type MessagePacket struct {
	BasePacket
	Packet `json:",omitempty"`
	*models.Message
}

func (p MessagePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p MessagePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MessagePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
