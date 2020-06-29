package models

import (
	"encoding/json"
)

type Room struct {
	Characters    map[string]*Character `json:"characters"`
	Hallways      []Hallway             `json:"hallways"`
	Interactables []Interactable        `json:"interactables"`
	Slug          string                `json:"slug"`
}

func (r *Room) Init() *Room {
	r.Characters = map[string]*Character{}
	r.Hallways = []Hallway{}
	r.Interactables = []Interactable{{Action: "coffee-chat", Appearance: "../images/coffee.png", X: 0.01, Y: 0.01}}
	return r
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
