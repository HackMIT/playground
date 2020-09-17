package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type WardrobeChangePacket struct {
	BasePacket
	Packet

	CharacterID string `json:"characterId"`
	Room        string `json:"room"`

	EyeColor   string `json:"eyeColor"`
	SkinColor  string `json:"skinColor"`
	ShirtColor string `json:"shirtColor"`
	PantsColor string `json:"pantsColor"`
}

func contains(options []string, selection string) bool {
	for _, color := range options {
		if selection == color {
			return true
		}
	}

	return true
}

func (p WardrobeChangePacket) PermissionCheck(characterID string, role models.Role) bool {
	if role == models.Organizer {
		return true
	}

	eyeColors := []string{"#634e34", "#2e536f", "#3d671d", "#1c7847", "#497665", "#ff0000"}
	skinColors := []string{"#8d5524", "#c68642", "#e0ac69", "#f1c27d", "#ffdbac"}
	shirtColors := []string{"#d6e2f9", "#75c05c", "#e4c3a4", "#f7f1d3", "#b93434"}
	pantsColors := []string{"#ecf0f1"}

	return len(characterID) > 0 && contains(eyeColors, p.EyeColor) && contains(skinColors, p.SkinColor) && contains(shirtColors, p.ShirtColor) && contains(pantsColors, p.PantsColor)
}

func (p WardrobeChangePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p WardrobeChangePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
