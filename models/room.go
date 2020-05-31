package models

import (
	"encoding/json"
	"github.com/google/uuid"
)

type Room struct {
	Characters map[uuid.UUID]*Character `json:"characters"`
	Hallways   []Hallway                `json:"hallways"`
	Slug       string                   `json:"slug"`
}

func (r *Room) Init() *Room {
	r.Characters = map[uuid.UUID]*Character{}
	r.Hallways = []Hallway{}
	return r
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
