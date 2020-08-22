package models

import (
	"encoding/json"
)

type Sponsor struct {
	Name string `json:"name"`
	Id string `json:"id"`
	Color string `json:"color"`
	Hackerqueue []string `json:"hackerqueue"`
}

func (s Sponsor) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s Sponsor) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
