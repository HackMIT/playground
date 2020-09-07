package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a song is added to queue
type SongPacket struct {
	BasePacket
	Packet
	*models.Song
}

func (p *SongPacket) Init(song *models.Song) *SongPacket {
	p.BasePacket = BasePacket{Type: "song"}
	p.Song = song
	return p
}

func (p SongPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}

func (p SongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
