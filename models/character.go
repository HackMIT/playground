package models

import (
	"github.com/google/uuid"
)

// Character is the digital representation of a client
type Character struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	School   string  `json:"school"`
	GradYear int     `json:"gradYear"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Room     string  `json:"room"`
	Ingest   int     `json:"ingest"`
}

func NewCharacter(quillData map[string]interface{}) *Character {
	c := new(Character)
	c.ID = uuid.New().String()
	c.Name = quillData["profile"].(map[string]interface{})["name"].(string)
	c.School = quillData["profile"].(map[string]interface{})["school"].(string)
	// TODO: Fix this
	c.GradYear = 2022
	c.X = 0.5
	c.Y = 0.5
	c.Room = "home"
	return c
}
