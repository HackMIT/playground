package packet

import (
    "encoding/json"
)

type UpdateMapPacket struct {
    BasePacket
    Lat float64 `json:"lat"`
    Lng float64 `json:"lng"`
    Name string `json:"name"`
}

func (p UpdateMapPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p UpdateMapPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
