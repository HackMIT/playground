package packet

import (
	"encoding/json"
	"time"

	"github.com/techx/playground/db"
	"github.com/techx/playground/db/models"
	"github.com/techx/playground/utils"
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

type FriendUpdatePacket struct {
	BasePacket
	Packet

	Friend Friend `json:"friend"`
}

func NewFriendUpdatePacket(characterID string, friendID string) *FriendUpdatePacket {
	pip := db.GetInstance().Pipeline()
	friendCmd := pip.HGetAll("character:" + friendID)
	isTeammateCmd := pip.SIsMember("character:"+characterID+":teammates", friendID)
	isRequestCmd := pip.SIsMember("character:"+characterID+":requests", friendID)
	pip.Exec()

	friendData, _ := friendCmd.Result()
	friend := new(models.Character)
	utils.Bind(friendData, friend)

	isTeammate, _ := isTeammateCmd.Result()
	isRequest, _ := isRequestCmd.Result()

	return &FriendUpdatePacket{
		BasePacket: BasePacket{
			Type: "friend_update",
		},
		Friend: Friend{
			ID:       friendID,
			Name:     friend.Name,
			School:   friend.School,
			Status:   0,
			Teammate: isTeammate,
			Pending:  isRequest,
			LastSeen: time.Now(),
		},
	}
}

func (p FriendUpdatePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p FriendUpdatePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
