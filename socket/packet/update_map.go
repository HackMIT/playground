package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type UpdateMapPacket struct {
	BasePacket
	Packet           `json:",omitempty"`
	*models.Location `json:"location"`
}

func (p UpdateMapPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p UpdateMapPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p UpdateMapPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
