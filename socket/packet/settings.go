package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when settings are changed
type SettingsPacket struct {
	BasePacket
	Packet

	// The client's new settings
	Settings *models.Settings `json:"settings"`

	CheckTwitter bool `json:"checkTwitter"`
}

func (p SettingsPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p SettingsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SettingsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
