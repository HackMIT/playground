package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by server to update a hacker on their position in the queue
type QueueUpdateHackerPacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`
	Position  int    `json:"position"`
	URL       string `json:"url,omitempty"`

	// Server attributes
	CharacterIDs []string `json:"characterIds"`
}

func NewQueueUpdateHackerPacket(sponsorID string, position int, url string) *QueueUpdateHackerPacket {
	return &QueueUpdateHackerPacket{
		BasePacket: BasePacket{
			Type: "queue_update_hacker",
		},
		SponsorID: sponsorID,
		Position:  position,
		URL:       url,
	}
}

// This isn't needed -- remove later
func (p QueueUpdateHackerPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p QueueUpdateHackerPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueUpdateHackerPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
