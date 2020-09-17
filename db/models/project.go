package models

import "encoding/json"

type Project struct {
	Challenges  string `json:"challenges" redis:"challenges"`
	Emails      string `json:"emails" redis:"emails"`
	Name        string `json:"name" redis:"name"`
	Pitch       string `json:"pitch" redis:"pitch"`
	SubmittedAt int    `json:"submittedAt" redis:"submittedAt"`
	Track       string `json:"track" redis:"track"`
	Zoom        string `json:"zoom" redis:"zoom"`
}

func (p Project) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p Project) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
