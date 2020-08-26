package packet

import (
	"encoding/json"
	"time"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"

	"github.com/go-redis/redis/v7"
)

type Friend struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	School   string    `json:"school"`
	Status   int       `json:"status"`
	Teammate bool      `json:"teammate"`
	Pending  bool      `json:"pending"`
	LastSeen time.Time `json:"lastSeen"`
}

type FriendsPacket struct {
	BasePacket

	Friends []Friend `json:"friends"`
}

func NewFriendsPacket(characterID string) *FriendsPacket {
	pip := db.GetInstance().Pipeline()
	teammatesCmd := pip.SMembers("character:" + characterID + ":teammates")
	friendsCmd := pip.SMembers("character:" + characterID + ":friends")
	requestsCmd := pip.SMembers("character:" + characterID + ":requests")
	pip.Exec()

	teammateIDs, _ := teammatesCmd.Result()
	friendIDs, _ := friendsCmd.Result()
	requestIDs, _ := requestsCmd.Result()

	pip = db.GetInstance().Pipeline()
	teammateCmds := make([]*redis.StringStringMapCmd, len(teammateIDs))
	friendCmds := make([]*redis.StringStringMapCmd, len(friendIDs))
	requestCmds := make([]*redis.StringStringMapCmd, len(requestIDs))

	for i, id := range teammateIDs {
		teammateCmds[i] = pip.HGetAll("character:" + id)
	}

	for i, id := range friendIDs {
		friendCmds[i] = pip.HGetAll("character:" + id)
	}

	for i, id := range requestIDs {
		requestCmds[i] = pip.HGetAll("character:" + id)
	}

	pip.Exec()

	i := 0
	friends := make([]Friend, len(teammateIDs)+len(friendIDs)+len(requestIDs))

	for j, cmd := range teammateCmds {
		data, _ := cmd.Result()
		res := new(models.Character)
		utils.Bind(data, res)

		friends[i] = Friend{
			ID:       teammateIDs[j],
			Name:     res.Name,
			School:   res.School,
			Status:   0,
			Teammate: true,
			LastSeen: time.Now(),
		}

		i++
	}

	for j, cmd := range friendCmds {
		data, _ := cmd.Result()
		res := new(models.Character)
		utils.Bind(data, res)

		friends[i] = Friend{
			ID:       friendIDs[j],
			Name:     res.Name,
			School:   res.School,
			Status:   0,
			LastSeen: time.Now(),
		}

		i++
	}

	for j, cmd := range requestCmds {
		data, _ := cmd.Result()
		res := new(models.Character)
		utils.Bind(data, res)

		friends[i] = Friend{
			ID:       requestIDs[j],
			Name:     res.Name,
			School:   res.School,
			Status:   0,
			Pending:  true,
			LastSeen: time.Now(),
		}

		i++
	}

	return &FriendsPacket{
		BasePacket: BasePacket{
			Type: "friends",
		},
		Friends: friends,
	}
}

func (p FriendsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p FriendsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
