package packet

import (
	"encoding/json"
	"time"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a song is added to queue
type SongPacket struct {
	BasePacket
	*models.Song
	RequiresWarning bool `json:"requiresWarning"`
	Remove bool `json:"remove"`
	Playing bool `json:"playing"`
	Start int `json:"start"`
	EndTime time.Time `json:"endTime"`
}

func (p *SongPacket) Init(song *models.Song) *SongPacket {
	p.BasePacket = BasePacket{Type: "song"}
	p.Song = song
	p.RequiresWarning = false
	p.Remove = false
	p.Playing = false
	return p
}

func (p SongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
