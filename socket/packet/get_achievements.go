package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type GetAchievementsPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	ID string `json:"id"`
}

func (p GetAchievementsPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p GetAchievementsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetAchievementsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
