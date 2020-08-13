package models

import (
	"encoding/json"
)

type Room struct {
	Characters    map[string]*Character `json:"characters"`
	Elements      map[string]*Element   `json:"elements"`
	Hallways      map[string]*Hallway             `json:"hallways"`
	Interactables []*Interactable       `json:"interactables"`
	Slug          string                `json:"slug"`
	Sponsor       bool                  `json:"sponsor"`
}

func (r *Room) Init() *Room {
	r.Characters = map[string]*Character{}
	r.Elements = map[string]*Element{}
	r.Hallways = map[string]*Hallway{}
	r.Interactables = []*Interactable{}
	r.Sponsor = false
	return r
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
