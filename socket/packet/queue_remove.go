package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by hackers to take themselves off queue
type QueueRemovePacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`

	CharacterID string `json:"characterId"`
}

func (p QueueRemovePacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p QueueRemovePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueRemovePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
