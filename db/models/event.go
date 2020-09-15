package models

import "time"

type Event struct {
	Name      string     `json:"name" redis:"name"`
	StartTime *time.Time `json:"startTime" redis:"startTime"`

	// Duration of the event, in minutes
	Duration int    `json:"duration" redis:"duration"`
	Type     string `json:"type" redis:"type"`
}
