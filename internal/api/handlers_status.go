package api

import (
	"encoding/json"
	"log"
	"net/http"
	"status-page/internal/db"
	"time"
)

// ---- Response types ----

type PublicStatusResponse struct {
	Projects []PublicProjectData `json:"projects"`
	Status   string              `json:"status"` // "All Systems Operational" | "Partial Outage" | "Major Outage"
}

type PublicProjectData struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Components []ComponentData `json:"components"`
}

type ComponentData struct {
	Name      string        `json:"name"`      // "Frontend", "Backend", "Bot", ...
	Type      string        `json:"type"`      // component_type from DB
	IsUp      bool          `json:"is_up"`
	UptimePct float64       `json:"uptime_pct"`
	History   []DailyStatus `json:"history"` // 7 kunlik
}

type DailyStatus struct {
	Date string `json:"date"`
	IsUp bool   `json:"is_up"`
}

// ---- Handler ----

func GetPublicStatusHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[STATUS] Public status requested")
	w.Header().Set("Content-Type", "application/json")

	// 1. Loyihalarni olish
	rows, err := db.DB.Query(`SELECT id, name, slug FROM projects WHERE show_on_public_page = 1 ORDER BY id`)
	if err != nil {
		log.Printf("[STATUS] ERROR querying projects: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	projectsMap := make(map[int]*PublicProjectData)
	var projectOrder []int

	for rows.Next() {
		var p PublicProjectData
		var slug string
		if err := rows.Scan(&p.ID, &p.Name, &slug); err != nil {
			log.Printf("[STATUS] ERROR scanning project: %v", err)
			continue
		}
		p.Components = []ComponentData{}
		projectsMap[p.ID] = &p
		projectOrder = append(projectOrder, p.ID)
	}
	log.Printf("[STATUS] Loaded %d projects", len(projectsMap))

	// 2. Monitorlarni olish
	monitors, err := db.GetAllMonitors()
	if err != nil {
		log.Printf("[STATUS] ERROR getting monitors: %v", err)
	}
	log.Printf("[STATUS] Loaded %d monitors", len(monitors))

	allUp := true

	for _, m := range monitors {
		if m.ProjectID == nil || !m.ShowOnPublicPage {
			log.Printf("[STATUS]   SKIP monitor id=%d (no project or not public)", m.ID)
			continue
		}
		proj, exists := projectsMap[*m.ProjectID]
		if !exists {
			log.Printf("[STATUS]   SKIP monitor id=%d (project %d not found)", m.ID, *m.ProjectID)
			continue
		}

		// 7 kunlik history
		histRows, err := db.DB.Query(`
			SELECT is_up, date(checked_at) as day
			FROM heartbeats
			WHERE monitor_id = ? AND checked_at >= date('now', '-7 days')
			ORDER BY checked_at DESC
		`, m.ID)

		var rawHistory []DailyStatus
		if err == nil {
			for histRows.Next() {
				var d DailyStatus
				if err := histRows.Scan(&d.IsUp, &d.Date); err == nil {
					rawHistory = append(rawHistory, d)
				}
			}
			histRows.Close()
		}

		// Kunlar bo'yicha aggregate
		dailyMap := make(map[string]bool)
		for _, rh := range rawHistory {
			if cur, ok := dailyMap[rh.Date]; ok {
				if !rh.IsUp {
					dailyMap[rh.Date] = false
				} else {
					dailyMap[rh.Date] = cur
				}
			} else {
				dailyMap[rh.Date] = rh.IsUp
			}
		}

		// So'nggi 7 kun
		var history []DailyStatus
		upCount := 0
		for i := 6; i >= 0; i-- {
			dateStr := time.Now().AddDate(0, 0, -i).Format("2006-01-02")
			isUp := true
			if val, ok := dailyMap[dateStr]; ok {
				isUp = val
			}
			if isUp {
				upCount++
			}
			history = append(history, DailyStatus{Date: dateStr, IsUp: isUp})
		}
		uptimePct := float64(upCount) / 7.0 * 100.0

		// Hozirgi status (eng oxirgi heartbeat)
		isCurrentlyUp := true
		if len(rawHistory) > 0 {
			isCurrentlyUp = rawHistory[0].IsUp
		}
		if !isCurrentlyUp {
			allUp = false
		}

		// Component nomi
		compName := capitalize(m.ComponentType)
		if compName == "" {
			compName = "Service"
		}

		proj.Components = append(proj.Components, ComponentData{
			Name:      compName,
			Type:      m.ComponentType,
			IsUp:      isCurrentlyUp,
			UptimePct: uptimePct,
			History:   history,
		})
	}

	// 3. Response yasash
	response := PublicStatusResponse{}
	for _, id := range projectOrder {
		if p, ok := projectsMap[id]; ok {
			response.Projects = append(response.Projects, *p)
		}
	}
	if response.Projects == nil {
		response.Projects = []PublicProjectData{}
	}

	downCount := 0
	for _, p := range response.Projects {
		for _, c := range p.Components {
			if !c.IsUp {
				downCount++
			}
		}
	}
	if allUp || downCount == 0 {
		response.Status = "All Systems Operational"
	} else if downCount > 2 {
		response.Status = "Major Outage"
	} else {
		response.Status = "Partial Outage"
	}

	log.Printf("[STATUS] Response: %d projects, status=%s", len(response.Projects), response.Status)
	json.NewEncoder(w).Encode(response)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string([]byte{s[0] - 32}) + s[1:]
}
