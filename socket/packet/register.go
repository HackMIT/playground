package packet

import (
	"encoding/json"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/techx/playground/db/models"
)

type RegisterPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	Name                string                `json:"name"`
	Location            string                `json:"location"`
	Bio                 string                `json:"bio"`
	PhoneNumber         string                `json:"phoneNumber"`
	BrowserSubscription *webpush.Subscription `json:"browserSubscription"`
}

func (p RegisterPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p RegisterPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p RegisterPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
