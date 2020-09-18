package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type ReportPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	CharacterID string `json:"characterId"`
	Text        string `json:"text"`
}

func (p ReportPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p ReportPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ReportPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
