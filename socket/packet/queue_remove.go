package packet

import (
	"encoding/json"
)

// Sent by hackers to take themselves off queue
type QueueRemovePacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`

	CharacterID string `json:"characterId"`
}

func (p QueueRemovePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueRemovePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}