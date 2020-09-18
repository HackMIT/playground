package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

// Sent by ingests when a user opens the jukebox for the first time
type JukeboxWarningPacket struct {
	BasePacket
	Packet `json:",omitempty"`
}

func NewJukeboxWarningPacket() *JukeboxWarningPacket {
	return &JukeboxWarningPacket{
		BasePacket: BasePacket{
			Type: "jukebox_warning",
		},
	}
}

func (p *JukeboxWarningPacket) Init() *JukeboxWarningPacket {
	p.BasePacket = BasePacket{Type: "jukebox_warning"}
	return p
}

func (p JukeboxWarningPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p JukeboxWarningPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p JukeboxWarningPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
