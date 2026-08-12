package api

import (
	"encoding/json"
	"log"
	"net/http"
	"status-page/internal/db"
	"time"
)

type PublicStatusResponse struct {
	Projects []PublicProjectData `json:"projects"`
	Status   string              `json:"status"` // "All Systems Operational" or "Partial Outage"
}

type PublicProjectData struct {
	ID          int               `json:"id"`
	Name        string            `json:"name"`
	Frontend    *ComponentStatus  `json:"frontend,omitempty"`
	Backend     *ComponentStatus  `json:"backend,omitempty"`
}

type ComponentStatus struct {
	IsUp       bool               `json:"is_up"`
	UptimePct  float64            `json:"uptime_pct"`
	History    []DailyHistory     `json:"history"` // 7 days
}

type DailyHistory struct {
	Date  string `json:"date"`
	IsUp  bool   `json:"is_up"`
}

func GetPublicStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[STATUS] Public status page requested")

	// 1. Fetch projects
	rows, err := db.DB.Query(`SELECT id, name, slug, description FROM projects WHERE show_on_public_page = 1`)
	if err != nil {
		log.Printf("[STATUS] ERROR: failed to query projects: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projectsMap = make(map[int]*PublicProjectData)
	var response PublicStatusResponse
	response.Status = "All Systems Operational"

	for rows.Next() {
		var p PublicProjectData
		var slug, desc string
		if err := rows.Scan(&p.ID, &p.Name, &slug, &desc); err != nil {
			log.Printf("[STATUS] ERROR: failed to scan project row: %v", err)
			continue
		}
		log.Printf("[STATUS] Found project: id=%d name=%s", p.ID, p.Name)
		projectsMap[p.ID] = &p
	}
	log.Printf("[STATUS] Total projects loaded: %d", len(projectsMap))

	// 2. Fetch monitors
	monitors, err := db.GetAllMonitors()
	if err != nil {
		log.Printf("[STATUS] ERROR: failed to get monitors: %v", err)
	}
	log.Printf("[STATUS] Total monitors loaded: %d", len(monitors))
	allUp := true

	for _, m := range monitors {
		log.Printf("[STATUS] Monitor: id=%d name=%s project_id=%v component=%s show_public=%v",
			m.ID, m.Name, m.ProjectID, m.ComponentType, m.ShowOnPublicPage)

		if m.ProjectID == nil || !m.ShowOnPublicPage {
			log.Printf("[STATUS]   -> SKIPPED (no project_id or not public)")
			continue
		}
		proj, exists := projectsMap[*m.ProjectID]
		if !exists {
			log.Printf("[STATUS]   -> SKIPPED (project %d not found in public projects)", *m.ProjectID)
			continue
		}
		log.Printf("[STATUS]   -> Assigned to project '%s'", proj.Name)

		// Fetch history for 7 days
		// We'll fetch all within last 7 days and aggregate by day
		historyRows, err := db.DB.Query(`
			SELECT is_up, date(checked_at) as day 
			FROM heartbeats 
			WHERE monitor_id = ? AND checked_at >= date('now', '-7 days')
			ORDER BY checked_at DESC
		`, m.ID)
		
		var rawHistory []DailyHistory
		if err == nil {
			for historyRows.Next() {
				var dh DailyHistory
				if err := historyRows.Scan(&dh.IsUp, &dh.Date); err == nil {
					rawHistory = append(rawHistory, dh)
				}
			}
			historyRows.Close()
		}

		// Aggregate by day (if any down, day is down)
		dailyMap := make(map[string]bool)
		for _, rh := range rawHistory {
			if _, ok := dailyMap[rh.Date]; ok {
				if !rh.IsUp {
					dailyMap[rh.Date] = false // Once down, day is down
				}
			} else {
				dailyMap[rh.Date] = rh.IsUp
			}
		}

		// Generate exactly 7 days
		var aggregatedHistory []DailyHistory
		upDaysCount := 0
		for i := 6; i >= 0; i-- {
			// Calculate the date string for (today - i days)
			// In Go, formatting time:
			dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			
			isUp := true
			if val, ok := dailyMap[dateStr]; ok {
				isUp = val
			} else {
				// No data for this day, assume up or ignore
				isUp = true
			}
			
			if isUp {
				upDaysCount++
			}
			
			aggregatedHistory = append(aggregatedHistory, DailyHistory{
				Date: dateStr,
				IsUp: isUp,
			})
		}

		uptimePct := (float64(upDaysCount) / 7.0) * 100.0

		// Current status
		isCurrentlyUp := true
		if len(rawHistory) > 0 {
			isCurrentlyUp = rawHistory[0].IsUp
		}

		if !isCurrentlyUp {
			allUp = false
		}

		compStatus := &ComponentStatus{
			IsUp:      isCurrentlyUp,
			UptimePct: uptimePct,
			History:   aggregatedHistory,
		}

		if m.ComponentType == "frontend" {
			proj.Frontend = compStatus
		} else {
			proj.Backend = compStatus
		}
	}

	for _, p := range projectsMap {
		response.Projects = append(response.Projects, *p)
	}

	if !allUp && len(response.Projects) > 0 {
		response.Status = "Partial Outage"
	}
	if response.Projects == nil {
		response.Projects = []PublicProjectData{}
	}

	json.NewEncoder(w).Encode(response)
}
