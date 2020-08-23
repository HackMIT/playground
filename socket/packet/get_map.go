package packet

import (
	"encoding/json"
)

type GetMapPacket struct {
	BasePacket
}

func (p GetMapPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetMapPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
