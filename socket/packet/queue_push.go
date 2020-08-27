package packet

import (
	"encoding/json"
	
	"github.com/techx/playground/db/models"
)

// Sent by hackers to push themselves onto queue
type QueuePushPacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`

	Character *models.Character `json:"character"`
}

func (p QueuePushPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueuePushPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
