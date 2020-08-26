package packet

import (
	"encoding/json"
)

// Sent by sponsors to pop first hacker from queue
type QueuePopPacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`

	CharacterID string `json:"characterId"`
}

func (p QueuePopPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueuePopPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
