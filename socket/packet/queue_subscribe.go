package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// sent by hackers and sponsors to subscribe to queue updates
type QueueSubscribePacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`

	Characters []*models.Character `json:"characters"`
}

func (p QueueSubscribePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p QueueSubscribePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueSubscribePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
