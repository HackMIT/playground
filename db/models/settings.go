package models

import (
	"encoding/json"
)

type Settings struct {
	MusicMuted bool `json:"musicMuted" redis:"musicMuted"`
	SoundMuted bool `json:"soundMuted" redis:"soundMuted"`
	TwitterHandle string `json:"twitterHandle" redis:"twitterHandle"`
}

func (s Settings) MarshalBinary() ([]byte, error) {
	return json.Marshal(s)
}

func (s Settings) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, s)
}
