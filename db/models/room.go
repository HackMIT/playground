package models

import (
	"encoding/json"
)

type Room struct {
	Characters map[string]*Character `json:"characters" redis:"-"`
	Elements   map[string]*Element   `json:"elements" redis:"-"`
	Hallways   map[string]*Hallway   `json:"hallways" redis:"-"`
	Slug       string                `json:"slug" redis:"slug"`
	Sponsor    bool                  `json:"sponsor" redis:"sponsor"`
}

func (r *Room) Init() *Room {
	r.Characters = map[string]*Character{}
	r.Elements = map[string]*Element{}
	r.Hallways = map[string]*Hallway{}
	r.Sponsor = false
	return r
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
