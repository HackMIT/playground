package packet

import (
	"encoding/json"
)

type FriendRequestPacket struct {
	BasePacket

	SenderID    string `json:"senderId"`
	RecipientID string `json:"recipientId"`
}

func (p FriendRequestPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p FriendRequestPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
