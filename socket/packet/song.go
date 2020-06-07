package packet

import (
	"encoding/json"

	"github.com/techx/playground/models"
)

// Sent by ingests when a song is added to queue
type SongPacket struct {
	BasePacket

	// The added song
	Song *models.Song `json:"song"`
}

func (p *SongPacket) Init(song *models.Song) *SongPacket {
	p.BasePacket = BasePacket{Type: "song"}
	p.Song = song
	return p
}

func (p SongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
