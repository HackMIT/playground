package packet

import "github.com/techx/playground/db/models"

type GetAchievementsPacket struct {
	BasePacket
}

func (p GetAchievementsPacket) PermissionCheck(characterID string, role models.Role) bool {
	return true
}
