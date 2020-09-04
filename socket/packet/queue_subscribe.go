package packet

import (
	"encoding/json"

	"github.com/go-redis/redis/v7"
	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"
)

// sent by hackers and sponsors to subscribe to queue updates
type QueueSubscribePacket struct {
	BasePacket

	SponsorID string `json:"sponsorId"`

	Characters []*models.Character `json:"characters"`
}

func NewQueueSubscribePacket(sponsorID string) *QueueSubscribePacket {
	p := QueueSubscribePacket{
		BasePacket: BasePacket{
			Type: "queue_subscribe",
		},
		SponsorID: sponsorID,
	}

	hackerIDs, _ := db.GetInstance().LRange("sponsor:"+sponsorID+":hackerqueue", 0, -1).Result()

	pip := db.GetInstance().Pipeline()
	characterCmds := make([]*redis.StringStringMapCmd, len(hackerIDs))
	characters := make([]*models.Character, len(characterCmds))

	for i, hackerID := range hackerIDs {
		characterCmds[i] = pip.HGetAll("character:" + hackerID)
		characters[i] = new(models.Character)
	}

	pip.Exec()

	for i, characterCmd := range characterCmds {
		characterRes, _ := characterCmd.Result()
		utils.Bind(characterRes, characters[i])
		characters[i].ID = hackerIDs[i]
	}

	p.Characters = characters

	return &p
}

func (p QueueSubscribePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p QueueSubscribePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
