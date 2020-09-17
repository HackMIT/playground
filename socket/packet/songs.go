package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type SongsPacket struct {
	BasePacket

	Songs  []*models.Song `json:"songs"`
}

func NewSongsPacket(songs []*models.Song) *SongsPacket {
	return &SongsPacket{
		BasePacket: BasePacket{
			Type: "songs",
		},
		Songs:  songs,
	}
}

func (p SongsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SongsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}