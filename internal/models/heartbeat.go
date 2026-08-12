package models

import "time"

type Heartbeat struct {
	ID             int       `json:"id"`
	MonitorID      int       `json:"monitor_id"`
	IsUp           bool      `json:"is_up"`
	ResponseTimeMs int       `json:"response_time_ms"`
	StatusCode     int       `json:"status_code"`
	ErrorMessage   string    `json:"error_message"`
	CheckedAt      time.Time `json:"checked_at"`
}
