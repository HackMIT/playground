package packet

import (
    "encoding/json"
)

type MessagesPacket struct {
    BasePacket

    Messages []map[string]string `json:"messages"`
    Recipient string `json:"recipient"`
}

func NewMessagesPacket(messages []map[string]string, recipient string) *MessagesPacket {
    return &MessagesPacket{
        BasePacket: BasePacket{
            Type: "messages",
        },
        Messages: messages,
        Recipient: recipient,
    }
}

func (p MessagesPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MessagesPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
