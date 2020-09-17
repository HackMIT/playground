package models

import (
	"encoding/json"
)

type Sponsor struct {
	Challenges  string `json:"challenges" redis:"challenges"`
	Description string `json:"description" redis:"description"`
	URL         string `json:"url" redis:"url"`
	Name        string `json:"name" redis:"name"`
	ID          string `json:"id" redis:"-"`
	QueueOpen   bool   `json:"queueOpen" redis:"queueOpen"`
}

func (s Sponsor) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s Sponsor) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
