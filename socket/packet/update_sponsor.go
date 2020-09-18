package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type UpdateSponsorPacket struct {
	BasePacket
	Packet `json:",omitempty"`
	*models.Sponsor

	SetQueueOpen bool `json:"setQueueOpen"`
}

func (p UpdateSponsorPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && (role == models.SponsorRep || role == models.Organizer)
}

func (p UpdateSponsorPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p UpdateSponsorPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
