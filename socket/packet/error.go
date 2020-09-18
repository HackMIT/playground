package packet

import (
	"encoding/json"
)

type ErrorPacket struct {
	BasePacket
	Packet `json:",omitempty"`

	Code int `json:"code"`
}

func NewErrorPacket(code int) *ErrorPacket {
	p := new(ErrorPacket)
	p.BasePacket = BasePacket{Type: "error"}
	p.Code = code
	return p
}

func (p ErrorPacket) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p ErrorPacket) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
