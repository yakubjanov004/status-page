package models

import "time"

type MaintenanceLog struct {
	ID              int        `json:"id"`
	EventType       string     `json:"event_type"`
	Description     string     `json:"description"`
	StartedAt       time.Time  `json:"started_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
}
