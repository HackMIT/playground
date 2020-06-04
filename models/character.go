package models

import (
	"github.com/techx/playground/config"
	"github.com/google/uuid"
)

// Character is the digital representation of a client
type Character struct {
	Id   string  `json:"id"`
	Name string  `json:"name"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Room string  `json:"room"`
}

func (c *Character) Init(id uuid.UUID, name string) *Character {
	config := config.GetConfig()

	c.Id = id.String()
	c.Name = name
	c.X = config.GetFloat64("character.start_x_pos")
	c.Y = config.GetFloat64("character.start_y_pos")
	c.Room = "home"

	return c
}
