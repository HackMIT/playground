package models

import (
	"encoding/json"
)

type Song struct {
	Artist string `json:"artist"`
	Duration int `json:"duration"`
	Name string `json:"name"`
	ThumbnailURL string `json:"thumbnailurl"`
	VidCode string `json:"vidcode"`
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