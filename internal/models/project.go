package models

import "time"

type Project struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Slug               string    `json:"slug"`
	Description        string    `json:"description"`
	ShowOnPublicPage   bool      `json:"show_on_public_page"`
	CreatedAt          time.Time `json:"created_at"`
	
	// Helper fields for UI (not mapped to DB directly)
	FrontendMonitor *Monitor `json:"frontend_monitor,omitempty"`
	BackendMonitor  *Monitor `json:"backend_monitor,omitempty"`
}
