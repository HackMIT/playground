package models

import (
	"encoding/json"
	"time"
)

type QueueStatus struct {
	CurrentSong Song `json:"currentsong"`
	SongEnd time.Time `json:"songend"`
}

func (q *QueueStatus) Init() *QueueStatus {
	return q
}

func (q QueueStatus) MarshalBinary() ([]byte, error) {
	return json.Marshal(q)
}

func (q QueueStatus) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, q)
}