package packet

import(
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type GetCurrentSongPacket struct {
	BasePacket
}

func (p GetCurrentSongPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p GetCurrentSongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetCurrentSongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}