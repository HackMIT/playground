package packet

import (
	"encoding/json"
)

type GetSongsPacket struct {
	BasePacket
}

func (p SongPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p GetSongsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetSongsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}