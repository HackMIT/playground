package packet

import (
	"encoding/json"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"
)

type AchievementsPacket struct {
	BasePacket
	Packet              `json:",omitempty"`
	models.Achievements `json:"achievements"`

	// The id of the client who we're getting achievements for
	ID string `json:"id"`
}

func NewAchievementsPacket(characterID string) *AchievementsPacket {
	p := new(AchievementsPacket)
	p.BasePacket = BasePacket{
		Type: "achievements",
	}

	p.ID = characterID

	res, _ := db.GetInstance().HGetAll("character:" + characterID + ":achievements").Result()
	utils.Bind(res, &p.Achievements)

	return p
}

func (p AchievementsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p AchievementsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
