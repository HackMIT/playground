package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type ProjectFormPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	Challenges []string `json:"challenges"`
	Teammates  []string `json:"teammates"`
	*models.Project
}

func (p ProjectFormPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p ProjectFormPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ProjectFormPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
