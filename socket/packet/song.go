package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a song is added to queue
type SongPacket struct {
	BasePacket
	Packet `json:",omitempty"`
	*models.Song
	RequiresWarning bool `json:"requiresWarning"`
	Remove          bool `json:"remove"`
}

func (p *SongPacket) Init(song *models.Song) *SongPacket {
	p.BasePacket = BasePacket{Type: "song"}
	p.Song = song
	p.RequiresWarning = false
	p.Remove = false
	return p
}

func (p SongPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0 && (!p.Remove || role == models.Organizer)
}

func (p SongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p SongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
