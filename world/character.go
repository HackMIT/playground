package world

import (
	"github.com/google/uuid"
)

// Character is the digital representation of a client
type Character struct {
	id string `json:"id"`
	Name string `json:"name"`
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

func newCharacter(id uuid.UUID, name string) *Character {
	return &Character{
		id: id.String(),
		Name: name,
		X: 0.5,
		Y: 0.5,
	}
}
