package packet

import (
	"encoding/json"
)

type GetSongsPacket struct {
	BasePacket
}

func (p GetSongsPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p GetSongsPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}