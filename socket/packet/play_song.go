package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a song is added to queue
type PlaySongPacket struct {
	BasePacket
	Packet
	*models.Song
	Start int `json:"start"`
	End int `json:"end"`
}

func (p *PlaySongPacket) Init(song *models.Song) *PlaySongPacket {
	p.BasePacket = BasePacket{Type: "playSong"}
	p.Song = song
	return p
}

func (p PlaySongPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p PlaySongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p PlaySongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
