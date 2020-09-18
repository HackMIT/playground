package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by hackers to join a sponsor's queue
type QueueJoinPacket struct {
	BasePacket

	SponsorID string   `json:"sponsorId"`
	Interests []string `json:"interests"`
}

func (p QueueJoinPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && (role == models.Hacker || role == models.Organizer)
}

func (p QueueJoinPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueJoinPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
