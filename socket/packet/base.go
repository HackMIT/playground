package packet

import (
	"github.com/techx/playground/db/models"
)

type Packet interface {
	PermissionCheck(characterID string, role models.Role) bool
}

// The base packet that can be sent between clients and server. These are all
// of the required attributes of any packet
type BasePacket struct {
	Packet

	// Identifies the type of packet
	Type string `json:"type"`
}

func (p *BasePacket) PermissionCheck(characterID string, role models.Role) bool {
	return false
}
