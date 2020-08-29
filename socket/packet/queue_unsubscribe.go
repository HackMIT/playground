package packet

import (
	"encoding/json"
)

// sent by hackers and sponsors to unsubscribe to queue updates
type QueueUnsubscribePacket struct {
	BasePacket 

	SponsorID string `json:"sponsorId"`
}

func (p QueueUnsubscribePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueUnsubscribePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}