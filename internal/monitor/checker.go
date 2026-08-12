package monitor

import (
	"crypto/tls"
	"net"
	"net/http"
	"status-page/internal/models"
	"strings"
	"time"
)

// Check — Kuma uslubida: default DOWN, muvaffaqiyatli bo'lsa UP
func Check(m *models.Monitor) *models.Heartbeat {
	hb := &models.Heartbeat{
		MonitorID: m.ID,
		CheckedAt: time.Now().UTC(), // Doimo UTC!
		IsUp:      false,            // Default: DOWN (Kuma kabi)
	}

	start := time.Now()
	timeout := time.Duration(m.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	switch m.Type {
	case "http":
		client := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			// Redirect qilmasin — faqat birinchi response ni olamiz
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		}

		url := m.URL
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
		}

		resp, err := client.Get(url)
		hb.Latency = int(time.Since(start).Milliseconds())

		if err != nil {
			hb.Message = err.Error()
			return hb
		}
		defer resp.Body.Close()

		hb.StatusCode = resp.StatusCode

		// Status code tekshirish: 200-399 oraligi muvaffaqiyatli
		if m.ExpectedStatusCode > 0 {
			if resp.StatusCode == m.ExpectedStatusCode {
				hb.IsUp = true
			} else {
				hb.Message = "Unexpected status code"
			}
		} else {
			// Default: 200-399 = UP
			if resp.StatusCode >= 200 && resp.StatusCode < 400 {
				hb.IsUp = true
			} else {
				hb.Message = "Unexpected status code"
			}
		}

	case "tcp":
		conn, err := net.DialTimeout("tcp", m.URL, timeout)
		hb.Latency = int(time.Since(start).Milliseconds())

		if err != nil {
			hb.Message = err.Error()
			return hb
		}
		conn.Close()
		hb.IsUp = true
	}

	return hb
}
