package models

import "encoding/json"

type QueueSubscriber struct {
	ID       string `json:"id"`
	Name     string `json:"name" redis:"name"`
	School   string `json:"school" redis:"school"`
	GradYear int    `json:"gradYear" redis:"gradYear"`
	Reason   int    `json:"reason" redis:"reason"`
}

func NewQueueSubscriber(c *Character) *QueueSubscriber {
	return &QueueSubscriber{
		ID:       c.ID,
		Name:     c.Name,
		School:   c.School,
		GradYear: c.GradYear,
		Reason:   0,
	}
}

func (s *QueueSubscriber) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s *QueueSubscriber) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
