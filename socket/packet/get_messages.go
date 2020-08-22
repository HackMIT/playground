package packet

import (
    "encoding/json"
)

type GetMessagesPacket struct {
    BasePacket

    Recipient string `json:"recipient"`
}

func (p GetMessagesPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetMessagesPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
