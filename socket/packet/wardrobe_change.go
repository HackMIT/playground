package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type WardrobeChangePacket struct {
	BasePacket
	Packet `json:",omitempty"`

	CharacterID string `json:"characterId"`
	Room        string `json:"room"`

	EyeColor   string `json:"eyeColor"`
	SkinColor  string `json:"skinColor"`
	ShirtColor string `json:"shirtColor"`
	PantsColor string `json:"pantsColor"`
}

func (p WardrobeChangePacket) PermissionCheck(characterID string, role models.Role) bool {
	if role == models.Organizer {
		return true
	}

	return len(characterID) > 0
}

func (p WardrobeChangePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p WardrobeChangePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
