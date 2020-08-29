package models

import "time"

type EventType int

const (
	MiniEvent EventType = iota
	Workshop
	Talk
)

type Event struct {
	Name      string     `json:"name" redis:"name"`
	StartTime *time.Time `json:"startTime" redis:"startTime"`

	// Duration of the event, in minutes
	Duration int `json:"duration" redis:"duration"`

	// TODO: How can we detect EventType in bind.go?
	Type EventType `json:"type" redis:"type"`
}
