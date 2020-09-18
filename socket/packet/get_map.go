package packet

import "github.com/techx/playground/db/models"

type GetMapPacket struct {
	BasePacket
}

func (p GetMapPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}
