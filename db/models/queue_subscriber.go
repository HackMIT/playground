package models

import (
	"encoding/json"
	"strings"
)

type QueueSubscriber struct {
	ID        string `json:"id"`
	Name      string `json:"name" redis:"name"`
	School    string `json:"school" redis:"school"`
	GradYear  int    `json:"gradYear" redis:"gradYear"`
	Interests string `json:"interests" redis:"interests"`
}

func NewQueueSubscriber(c *Character, interests []string) *QueueSubscriber {
	return &QueueSubscriber{
		ID:        c.ID,
		Name:      c.Name,
		School:    c.School,
		GradYear:  c.GradYear,
		Interests: strings.Join(interests, ","),
	}
}

func (s *QueueSubscriber) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *QueueSubscriber) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
