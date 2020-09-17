package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by hackers to take themselves off queue
type QueueRemovePacket struct {
	BasePacket

	SponsorID   string `json:"sponsorId"`
	CharacterID string `json:"characterId"`
	Zoom        string `json:"zoom"`
}

func (p QueueRemovePacket) PermissionCheck(characterID string, role models.Role) bool {
	if len(characterID) == 0 {
		// User is not signed in
		return false
	}

	// Can remove if the hacker is removing themself, or if the sponsor is taking them off the queue
	return role == models.SponsorRep || characterID == p.CharacterID
}

func (p QueueRemovePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueRemovePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
