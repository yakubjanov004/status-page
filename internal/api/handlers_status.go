package api

import (
	"encoding/json"
	"net/http"
	"status-page/internal/db"
	"strings"
	"time"
)

type PublicStatusResponse struct {
	Projects []PublicProjectData `json:"projects"`
	Status   string             `json:"status"`
}

type PublicProjectData struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Components []ComponentData `json:"components"`
}

type ComponentData struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	IsUp      bool          `json:"is_up"`
	UptimePct float64       `json:"uptime_pct"`
	Latency   int           `json:"latency"`
	History   []DailyStatus `json:"history"`
}

type DailyStatus struct {
	Date  string  `json:"date"`
	IsUp  bool    `json:"is_up"`
	Pct   float64 `json:"pct"` // Kun ichidagi uptime %
}

func GetPublicStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// 1. Loyihalar
	rows, err := db.DB.Query(`SELECT id, name, slug FROM projects WHERE show_on_public_page = 1 ORDER BY id`)
	if err != nil {
		json.NewEncoder(w).Encode(PublicStatusResponse{Projects: []PublicProjectData{}, Status: "Error"})
		return
	}
	defer rows.Close()

	type projInfo struct {
		data PublicProjectData
	}
	var projectList []projInfo

	for rows.Next() {
		var p projInfo
		var slug string
		if err := rows.Scan(&p.data.ID, &p.data.Name, &slug); err != nil {
			continue
		}
		p.data.Components = []ComponentData{}
		projectList = append(projectList, p)
	}

	// 2. Monitorlar
	monitors, err := db.GetAllMonitors()
	if err != nil {
		monitors = nil
	}

	// Kuma uslubida overall status
	hasUp := false
	hasDown := false

	for _, m := range monitors {
		if m.ProjectID == nil || !m.ShowOnPublicPage {
			continue
		}

		// Loyiha index topamiz
		projIdx := -1
		for i, p := range projectList {
			if p.data.ID == *m.ProjectID {
				projIdx = i
				break
			}
		}
		if projIdx == -1 {
			continue
		}

		// Eng oxirgi heartbeat — hozirgi status
		var isCurrentlyUp bool
		var latency int
		err := db.DB.QueryRow(
			`SELECT is_up, response_time_ms FROM heartbeats WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 1`,
			m.ID,
		).Scan(&isCurrentlyUp, &latency)
		if err != nil {
			isCurrentlyUp = true // Hali heartbeat yo'q
			latency = 0
		}

		if isCurrentlyUp {
			hasUp = true
		} else {
			hasDown = true
		}

		// 7 kunlik history — heartbeat based uptime %
		history := buildHistory(m.ID)

		// Umumiy uptime % (7 kun)
		var totalUp, totalChecks int
		db.DB.QueryRow(`
			SELECT 
				COALESCE(SUM(CASE WHEN is_up = 1 THEN 1 ELSE 0 END), 0),
				COUNT(*)
			FROM heartbeats
			WHERE monitor_id = ? AND checked_at >= datetime('now', '-7 days')
		`, m.ID).Scan(&totalUp, &totalChecks)

		uptimePct := 100.0
		if totalChecks > 0 {
			uptimePct = float64(totalUp) / float64(totalChecks) * 100.0
		}

		compName := strings.ToUpper(m.ComponentType[:1]) + m.ComponentType[1:]

		projectList[projIdx].data.Components = append(projectList[projIdx].data.Components, ComponentData{
			Name:      compName,
			Type:      m.ComponentType,
			IsUp:      isCurrentlyUp,
			UptimePct: uptimePct,
			Latency:   latency,
			History:   history,
		})
	}

	// 3. Response — Kuma overallStatus logikasi
	response := PublicStatusResponse{Projects: []PublicProjectData{}}
	for _, p := range projectList {
		response.Projects = append(response.Projects, p.data)
	}

	switch {
	case hasDown && hasUp:
		response.Status = "Partially Degraded Service"
	case hasDown && !hasUp:
		response.Status = "Major Outage"
	default:
		response.Status = "All Systems Operational"
	}

	json.NewEncoder(w).Encode(response)
}

func buildHistory(monitorID int) []DailyStatus {
	var history []DailyStatus
	now := time.Now().UTC()

	for i := 6; i >= 0; i-- {
		dateStr := now.AddDate(0, 0, -i).Format("2006-01-02")

		var totalChecks, upChecks int
		db.DB.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_up = 1 THEN 1 ELSE 0 END), 0)
			FROM heartbeats
			WHERE monitor_id = ? AND date(checked_at) = ?
		`, monitorID, dateStr).Scan(&totalChecks, &upChecks)

		isUp := true
		pct := 100.0
		if totalChecks > 0 {
			pct = float64(upChecks) / float64(totalChecks) * 100.0
			if pct < 100.0 {
				isUp = false
			}
		}

		history = append(history, DailyStatus{Date: dateStr, IsUp: isUp, Pct: pct})
	}

	return history
}
