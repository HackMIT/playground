package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type GetSongsPacket struct {
	BasePacket
}

func (p GetSongsPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p GetSongsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetSongsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}