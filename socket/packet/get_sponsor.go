package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type GetSponsorPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	SponsorID string `json:"id"`
}

func (p GetSponsorPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p GetSponsorPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetSponsorPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
