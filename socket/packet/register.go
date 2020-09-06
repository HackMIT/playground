package packet

import (
	"encoding/json"

	webpush "github.com/SherClockHolmes/webpush-go"
)

type RegisterPacket struct {
	BasePacket

	BrowserSubscription *webpush.Subscription `json:"browserSubscription"`
}

func (p RegisterPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p RegisterPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
