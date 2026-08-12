package models

import "time"

type Monitor struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Type               string    `json:"type"` // "http", "tcp"
	URL                string    `json:"url"`
	IntervalSeconds    int       `json:"interval_seconds"`
	TimeoutSeconds     int       `json:"timeout_seconds"`
	ExpectedStatusCode int       `json:"expected_status_code"`
	IsActive           bool      `json:"is_active"`
	ShowOnPublicPage   bool      `json:"show_on_public_page"`
	ProjectID          *int      `json:"project_id"`
	ComponentType      string    `json:"component_type"`
	CreatedAt          time.Time `json:"created_at"`
}
