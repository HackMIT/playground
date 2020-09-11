package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Role int

const (
	Guest Role = iota
	Organizer
	SponsorRep
	Mentor
	Hacker
)

// Character is the digital representation of a client
type Character struct {
	ID             string  `json:"id" redis:"-"`
	Name           string  `json:"name" redis:"name"`
	School         string  `json:"school" redis:"school"`
	GradYear       int     `json:"gradYear" redis:"gradYear"`
	X              float64 `json:"x" redis:"x"`
	Y              float64 `json:"y" redis:"y"`
	Room           string  `json:"room" redis:"room"`
	Ingest         string  `json:"ingest" redis:"ingest"`
	FeedbackOpened bool    `json:"feedbackOpened" redis:"feedbackOpened"`
	Role           int     `json:"role" redis:"role"`

	// If this character is a sponsor rep, this is their company's ID
	SponsorID string `json:"sponsorId,omitempty" redis:"sponsorId"`
}

func NewCharacter(name string) *Character {
	c := new(Character)
	c.ID = uuid.New().String()
	c.Name = name
	c.GradYear = 2022
	c.X = 0.5
	c.Y = 0.5
	c.Room = "home"
	c.Role = int(Organizer)
	return c
}

func NewTIMCharacter() *Character {
	c := new(Character)
	c.ID = "tim"
	c.Name = "TIM the Beaver"
	c.School = "MIT"
	c.GradYear = 9999
	c.X = 0.5
	c.Y = 0.5
	c.Room = "home"
	c.Role = int(Organizer)
	return c
}

func NewCharacterFromQuill(quillData map[string]interface{}) *Character {
	c := new(Character)
	c.ID = uuid.New().String()
	c.Name = quillData["profile"].(map[string]interface{})["name"].(string)
	c.School = quillData["profile"].(map[string]interface{})["school"].(string)
	// TODO: Fix this
	c.GradYear = 2022
	c.X = 0.5
	c.Y = 0.5
	c.Room = "home"
	c.Role = int(Hacker)
	return c
}

func (c *Character) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Character) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
