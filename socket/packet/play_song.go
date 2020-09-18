package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a song is added to queue
type PlaySongPacket struct {
	BasePacket
	Packet `json:",omitempty"`
	Song   *models.Song `json:"song"`
	Start  int          `json:"start"`
	End    int          `json:"end"`
}

func NewPlaySongPacket(song *models.Song, start int) *PlaySongPacket {
	return &PlaySongPacket{
		BasePacket: BasePacket{
			Type: "play_song",
		},
		Song:  song,
		Start: start,
	}
}

func (p *PlaySongPacket) Init(song *models.Song) *PlaySongPacket {
	p.BasePacket = BasePacket{Type: "play_song"}
	p.Song = song
	return p
}

func (p PlaySongPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p PlaySongPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p PlaySongPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
