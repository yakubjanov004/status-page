package models

import "time"

// Heartbeat — har bir tekshiruv natijasi
type Heartbeat struct {
	ID        int       `json:"id"`
	MonitorID int       `json:"monitor_id"`
	IsUp      bool      `json:"is_up"`
	StatusCode int      `json:"status_code"`
	Latency   int       `json:"latency"` // milliseconds
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checked_at"`
}
