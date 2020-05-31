package models

import (
	"encoding/json"
	"github.com/google/uuid"
)

type Room struct {
	Background string                   `json:"background"`
	Characters map[uuid.UUID]*Character `json:"characters"`
	Hallways   []Hallway                `json:"hallways"`
	Slug       string                   `json:"slug"`
}

func NewRoom(background string, slug string) *Room {
	return &Room{
		Background: background,
		Characters: map[uuid.UUID]*Character{},
		Hallways: []Hallway{},
		Slug: slug,
	}
}

func (r Room) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r Room) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}
