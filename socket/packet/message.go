package packet

import (
    "encoding/json"
)

type MessagePacket struct {
    BasePacket

    Sender string `json:"sender"`

    Text string `json:"text"`

    Recipient string `json:"recipient"`
}

func (p MessagePacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p MessagePacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
