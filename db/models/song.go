package models

import (
	"encoding/json"
)

type Song struct {
	Duration     int    `json:"duration"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Title        string `json:"title"`
	VidCode      string `json:"vidCode"`
	ID			 string `json:"id"`
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
