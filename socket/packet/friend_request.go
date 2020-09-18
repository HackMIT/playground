package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type FriendRequestPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	SenderID    string `json:"senderId"`
	RecipientID string `json:"recipientId"`
}

func (p FriendRequestPacket) PermissionCheck(characterID string, role models.Role) bool {
	return len(characterID) > 0
}

func (p FriendRequestPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p FriendRequestPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
