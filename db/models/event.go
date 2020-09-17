package models

type Event struct {
	Name string `json:"name" redis:"name"`
	URL  string `json:"url" redis:"url"`

	// Start time of the event, as a Unix timestamp
	StartTime int `json:"startTime" redis:"startTime"`

	// Duration of the event, in minutes
	Duration int    `json:"duration" redis:"duration"`
	Type     string `json:"type" redis:"type"`
}
