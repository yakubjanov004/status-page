package model

import (
	"time"
)

// Service represents a monitored service in the webhook system.
type Service struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Event represents a single up/down event received via webhook.
type Event struct {
	ID        int64     `json:"id"`
	ServiceID int64     `json:"service_id"`
	Action    string    `json:"action"` // "up" or "down"
	Time      time.Time `json:"time"`
	Payload   string    `json:"payload,omitempty"` // JSON string of meta
	CreatedAt time.Time `json:"created_at"`
}

// Incident represents a downtime incident for a service.
type Incident struct {
	ID              int64      `json:"id"`
	ServiceID       int64      `json:"service_id"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	DurationSeconds *int64     `json:"duration_seconds,omitempty"`
	Status          string     `json:"status"` // "open" or "closed"
	CreatedAt       time.Time  `json:"created_at"`
}

// WebhookRequest is the JSON body for POST /api/v1/webhook.
type WebhookRequest struct {
	Service string                 `json:"service"`
	Action  string                 `json:"action"` // "up" or "down"
	Time    string                 `json:"time"`   // ISO8601 UTC
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// ServiceStatus is the response item for GET /api/v1/services.
type ServiceStatus struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	LastStatus         string  `json:"last_status"` // "up", "down", "unknown"
	LastSeen           *string `json:"last_seen"`   // ISO8601 or null
	OpenIncidentsCount int     `json:"open_incidents_count"`
}

// IncidentResponse is the response item for GET /api/v1/services/{name}/incidents.
type IncidentResponse struct {
	ID              int64   `json:"id"`
	ServiceName     string  `json:"service_name"`
	StartTime       string  `json:"start_time"`
	EndTime         *string `json:"end_time,omitempty"`
	DurationSeconds *int64  `json:"duration_seconds,omitempty"`
	Status          string  `json:"status"`
}

// UptimeResponse is the response for GET /api/v1/services/{name}/uptime.
type UptimeResponse struct {
	ServiceName          string  `json:"service_name"`
	Window               string  `json:"window"`
	UptimePercent        float64 `json:"uptime_percent"`
	TotalDowntimeSeconds int64   `json:"total_downtime_seconds"`
}

// ErrorResponse is a standard JSON error response.
type ErrorResponse struct {
	Error     string `json:"error"`
	RequestID string `json:"request_id,omitempty"`
}

// SuccessResponse is a standard JSON success response.
type SuccessResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// PaginatedIncidentsResponse wraps incident list with pagination info.
type PaginatedIncidentsResponse struct {
	Incidents []IncidentResponse `json:"incidents"`
	Total     int                `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}
