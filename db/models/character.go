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

	defaultEyeColor   = "#634e34"
	defaultSkinColor  = "#e0ac69"
	defaultShirtColor = "#d6e2f8"
	defaultPantsColor = "#ecf0f1"
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
	Email          string  `json:"email" redis:"email"`
	Role           int     `json:"role" redis:"role"`
	IsCollege      bool    `json:"isCollege" redis:"isCollege"`

	// If this character is in a queue, this is the sponsor ID of the queue they're in
	QueueID string `json:"queueId" redis:"queueId"`

	// If this character is a sponsor rep, this is their company's ID
	SponsorID string `json:"sponsorId,omitempty" redis:"sponsorId"`

	// This character's project, if they have one
	Project *Project `json:"project" redis:"-"`

	// Clothes
	EyeColor   string `json:"eyeColor" redis:"eyeColor"`
	SkinColor  string `json:"skinColor" redis:"skinColor"`
	ShirtColor string `json:"shirtColor" redis:"shirtColor"`
	PantsColor string `json:"pantsColor" redis:"pantsColor"`
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
	c.EyeColor = defaultEyeColor
	c.SkinColor = defaultSkinColor
	c.ShirtColor = defaultShirtColor
	c.PantsColor = defaultPantsColor
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
	c.EyeColor = defaultEyeColor
	c.SkinColor = defaultSkinColor
	c.ShirtColor = defaultShirtColor
	c.PantsColor = defaultPantsColor
	return c
}

func (c *Character) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Character) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
