package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by server to update a hacker on their position in the queue
type QueueUpdateSponsorPacket struct {
	BasePacket

	Subscribers []*models.QueueSubscriber `json:"subscribers"`

	// Server attributes
	CharacterIDs []string `json:"characterIds"`
}

func NewQueueUpdateSponsorPacket(subscribers []*models.QueueSubscriber) *QueueUpdateSponsorPacket {
	return &QueueUpdateSponsorPacket{
		BasePacket: BasePacket{
			Type: "queue_update_sponsor",
		},
		Subscribers: subscribers,
	}
}

// This isn't needed -- remove later
func (p QueueUpdateSponsorPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p QueueUpdateSponsorPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueUpdateSponsorPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
