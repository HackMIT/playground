package models

import (
	"encoding/json"
)

type Song struct {
	Duration     int    `json:"duration" redis:"duration"`
	ThumbnailURL string `json:"thumbnailUrl" redis:"thumbnailUrl"`
	Title        string `json:"title" redis:"title"`
	VidCode      string `json:"vidCode" redis:"vidCode"`
	ID			 string `json:"id" redis:"-"`
}

func (s *Song) Init() *Song {
	return s
}

func (s Song) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s Song) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
