package models

import (
	"encoding/json"
	"time"
)

type Log struct {
	CharacterID string `json:"characterId" redis:"characterId"`
	Message     string `json:"message" redis:"message"`
	Timestamp   int64  `json:"timestamp" redis:"timestamp"`
}

func NewLog(characterID, message string) *Log {
	return &Log{
		CharacterID: characterID,
		Message:     message,
		Timestamp:   time.Now().Unix(),
	}
}

func (c *Log) MarshalBinary() ([]byte, error) {
	return json.Marshal(c)
}

func (c *Log) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, c)
}
