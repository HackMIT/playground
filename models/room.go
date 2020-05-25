package models

import (
	"encoding/json"
)

type Room struct {
	Background string `json:"background" mapstructure:"background"`
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
