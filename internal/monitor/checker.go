package monitor

import (
	"crypto/tls"
	"net"
	"net/http"
	"status-page/internal/models"
	"strings"
	"time"
)

func Check(m *models.Monitor) *models.Heartbeat {
	hb := &models.Heartbeat{
		MonitorID: m.ID,
		CheckedAt: time.Now(),
		IsUp:      false,
	}

	start := time.Now()
	timeout := time.Duration(m.TimeoutSeconds) * time.Second

	switch m.Type {
	case "http":
		client := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		
		url := m.URL
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}

		resp, err := client.Get(url)
		duration := time.Since(start).Milliseconds()
		hb.ResponseTimeMs = int(duration)

		if err != nil {
			hb.ErrorMessage = err.Error()
			return hb
		}
		defer resp.Body.Close()

		hb.StatusCode = resp.StatusCode
		if resp.StatusCode == m.ExpectedStatusCode {
			hb.IsUp = true
		} else {
			hb.ErrorMessage = "Unexpected status code"
		}

	case "tcp":
		conn, err := net.DialTimeout("tcp", m.URL, timeout)
		duration := time.Since(start).Milliseconds()
		hb.ResponseTimeMs = int(duration)

		if err != nil {
			hb.ErrorMessage = err.Error()
			return hb
		}
		conn.Close()
		hb.IsUp = true

	case "ping":
		// Simple ping could be implemented, but for simplicity, we treat TCP dial on port 80/443 or similar as ping if it's an IP
		// In a real scenario, ICMP ping requires root privileges or raw sockets
		hb.ErrorMessage = "Ping type is not fully implemented yet, use TCP or HTTP"
	}

	return hb
}
