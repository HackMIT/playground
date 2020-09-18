package packet

import (
	"encoding/json"

	"github.com/techx/playground/db/models"
)

type MessagesPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	Messages  []*models.Message `json:"messages"`
	Recipient string            `json:"recipient"`
}

func NewMessagesPacket(messages []*models.Message, recipient string) *MessagesPacket {
	return &MessagesPacket{
		BasePacket: BasePacket{
			Type: "messages",
		},
		Messages:  messages,
		Recipient: recipient,
	}
}

func (p MessagesPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MessagesPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
