package api

import (
	"encoding/json"
	"log"
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
	History   []DailyStatus `json:"history"`
}

type DailyStatus struct {
	Date string `json:"date"`
	IsUp bool   `json:"is_up"`
}

func GetPublicStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	// 1. Loyihalar
	rows, err := db.DB.Query(`SELECT id, name, slug FROM projects WHERE show_on_public_page = 1 ORDER BY id`)
	if err != nil {
		log.Printf("[STATUS] ERROR projects: %v", err)
		json.NewEncoder(w).Encode(PublicStatusResponse{Projects: []PublicProjectData{}, Status: "Error"})
		return
	}
	defer rows.Close()

	type projInfo struct {
		data  PublicProjectData
		slug  string
	}
	var projectList []projInfo

	for rows.Next() {
		var p projInfo
		if err := rows.Scan(&p.data.ID, &p.data.Name, &p.slug); err != nil {
			continue
		}
		p.data.Components = []ComponentData{}
		projectList = append(projectList, p)
	}

	// 2. Monitorlar
	monitors, err := db.GetAllMonitors()
	if err != nil {
		log.Printf("[STATUS] ERROR monitors: %v", err)
	}

	anyDown := false

	for _, m := range monitors {
		if m.ProjectID == nil || !m.ShowOnPublicPage {
			continue
		}

		// Bu monitor qaysi loyihaga tegishli?
		var projIdx int = -1
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
		var isCurrentlyUp bool = true
		err := db.DB.QueryRow(
			`SELECT is_up FROM heartbeats WHERE monitor_id = ? ORDER BY checked_at DESC LIMIT 1`,
			m.ID,
		).Scan(&isCurrentlyUp)
		if err != nil {
			// Hali heartbeat yo'q — unknown status, default true
			isCurrentlyUp = true
		}

		if !isCurrentlyUp {
			anyDown = true
		}

		// 7 kunlik history
		history := buildHistory(m.ID)

		// Uptime %
		upDays := 0
		for _, d := range history {
			if d.IsUp {
				upDays++
			}
		}
		uptimePct := float64(upDays) / float64(len(history)) * 100.0

		compName := strings.ToUpper(m.ComponentType[:1]) + m.ComponentType[1:]

		projectList[projIdx].data.Components = append(projectList[projIdx].data.Components, ComponentData{
			Name:      compName,
			Type:      m.ComponentType,
			IsUp:      isCurrentlyUp,
			UptimePct: uptimePct,
			History:   history,
		})
	}

	// 3. Response
	response := PublicStatusResponse{Projects: []PublicProjectData{}}
	for _, p := range projectList {
		response.Projects = append(response.Projects, p.data)
	}

	if anyDown {
		// Nechta down ekanini tekshiramiz
		downCount := 0
		totalCount := 0
		for _, p := range response.Projects {
			for _, c := range p.Components {
				totalCount++
				if !c.IsUp {
					downCount++
				}
			}
		}
		if downCount >= totalCount/2 {
			response.Status = "Major Outage"
		} else {
			response.Status = "Partial Outage"
		}
	} else {
		response.Status = "All Systems Operational"
	}

	json.NewEncoder(w).Encode(response)
}

func buildHistory(monitorID int) []DailyStatus {
	var history []DailyStatus

	// 7 kun — har bir kun uchun eng yomon natijani olamiz
	for i := 6; i >= 0; i-- {
		dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")

		var totalChecks, downChecks int
		db.DB.QueryRow(`
			SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_up = 0 THEN 1 ELSE 0 END), 0)
			FROM heartbeats
			WHERE monitor_id = ? AND date(checked_at) = ?
		`, monitorID, dateStr).Scan(&totalChecks, &downChecks)

		isUp := true
		if totalChecks > 0 && downChecks > 0 {
			isUp = false // Agar 1 ta ham down bo'lsa, kun qizil
		}

		history = append(history, DailyStatus{Date: dateStr, IsUp: isUp})
	}

	return history
}
