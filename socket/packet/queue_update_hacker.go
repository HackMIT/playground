package packet

import (
	"encoding/json"
)

// Sent by server to update a hacker on their position in the queue
type QueueUpdateHackerPacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`
	Position  int    `json:"position"`
	URL       string `json:"url,omitempty"`
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

func (p QueueUpdateHackerPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueUpdateHackerPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
