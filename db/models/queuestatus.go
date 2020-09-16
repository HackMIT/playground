package models

import (
	"encoding/json"
)

type QueueStatus struct {
	CurrentSong *Song `json:"currentSong"`
	SongEnd int64 `json:"songend"`
}

func NewQueueStatus(song *Song, end int64) *QueueStatus {
	return &QueueStatus{
		CurrentSong: song,
		SongEnd: end,
	}
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
