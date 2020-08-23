package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by clients when settings are changed
type SettingsPacket struct {
	BasePacket

	// The client's new settings
	Settings *models.Settings `json:"settings"`
}

func (p SettingsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SettingsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
