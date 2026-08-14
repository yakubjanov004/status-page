package api

import (
	"encoding/json"
	"net/http"
	"status-page/internal/config"
	"status-page/internal/db"
	"status-page/internal/models"
	"strings"
	"time"
)

type PublicStatusResponse struct {
	SiteName        string              `json:"site_name"`
	Status          string              `json:"status"`
	LastUpdated     string              `json:"last_updated"`
	OverallUptime   float64             `json:"overall_uptime_pct"`
	TotalServices   int                 `json:"total_services"`
	ActiveIncidents int                 `json:"active_incidents"`
	Projects        []PublicProjectData     `json:"projects"`
	RecentOutages   []RecentOutage          `json:"recent_outages"`
	MaintenanceLogs []models.MaintenanceLog `json:"maintenance_logs"`
}

type PublicProjectData struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Slug       string          `json:"slug"`
	Components []ComponentData `json:"components"`
}

type Outage struct {
	Start           string `json:"start"`
	End             string `json:"end"`
	DurationSeconds int    `json:"duration_seconds"`
}

type RecentOutage struct {
	ServiceName     string `json:"service_name"`
	Start           string `json:"start"`
	End             string `json:"end"`
	DurationSeconds int    `json:"duration_seconds"`
	IsOngoing       bool   `json:"is_ongoing"`
}

type ComponentData struct {
	Name              string        `json:"name"`
	Type              string        `json:"type"`
	IsUp              bool          `json:"is_up"`
	UptimePct         float64       `json:"uptime_pct"`
	Latency           int           `json:"latency"`
	TotalDowntimeSecs int           `json:"total_downtime_secs"`
	TotalOutages      int           `json:"total_outages"`
	LongestOutageSecs int           `json:"longest_outage_secs"`
	CreatedAt         string        `json:"created_at"`
	History           []DailyStatus `json:"history"`
}

type DailyStatus struct {
	Date    string   `json:"date"`
	IsUp    bool     `json:"is_up"`
	Pct     float64  `json:"pct"`
	HasData bool     `json:"has_data"`
	Outages []Outage `json:"outages"`
}

// GetPublicStatusHandler — cfg orqali site_name va boshqa sozlamalar uzatiladi
func GetPublicStatusHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		siteName := cfg.SiteName
		if siteName == "" {
			siteName = "Server Status"
		}

		// 1. Loyihalar
		rows, err := db.DB.Query(`SELECT id, name, slug FROM projects WHERE show_on_public_page = 1 ORDER BY id`)
		if err != nil {
			json.NewEncoder(w).Encode(PublicStatusResponse{
				SiteName:    siteName,
				Projects:    []PublicProjectData{},
				Status:      "Error",
				LastUpdated: time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		defer rows.Close()

		type projInfo struct {
			data PublicProjectData
		}
		var projectList []projInfo

		for rows.Next() {
			var p projInfo
			if err := rows.Scan(&p.data.ID, &p.data.Name, &p.data.Slug); err != nil {
				continue
			}
			p.data.Components = []ComponentData{}
			projectList = append(projectList, p)
		}
		if err := rows.Err(); err != nil {
			projectList = nil
		}

		// 2. Monitorlar
		monitors, err := db.GetAllMonitors()
		if err != nil {
			monitors = nil
		}

		hasUp := false
		hasDown := false

		totalUptimeSum := 0.0
		totalUptimeCount := 0
		activeIncidents := 0

		// Barcha outagelar ro'yxati (so'nggi 7 kun)
		var allRecentOutages []RecentOutage

		for _, m := range monitors {
			if m.ProjectID == nil || !m.ShowOnPublicPage {
				continue
			}

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
				activeIncidents++
			}

			// 7 kunlik history
			history, totalSeconds, downtimeSeconds, rawOutages := buildHistory(m, 7)

			uptimePct := 100.0
			if totalSeconds > 0 {
				uptime := totalSeconds - downtimeSeconds
				if uptime < 0 {
					uptime = 0
				}
				uptimePct = float64(uptime) / float64(totalSeconds) * 100.0
			}

			totalUptimeSum += uptimePct
			totalUptimeCount++

			// Outage statistikasi
			totalDowntime := 0
			longestOutage := 0
			for _, o := range rawOutages {
				dur := o.DurationSeconds
				totalDowntime += dur
				if dur > longestOutage {
					longestOutage = dur
				}
				allRecentOutages = append(allRecentOutages, RecentOutage{
					ServiceName:     m.Name,
					Start:           o.Start,
					End:             o.End,
					DurationSeconds: o.DurationSeconds,
					IsOngoing:       false,
				})
			}

			// Hozir ham down bo'lsa — oxirgi outage'ni ongoing deb belgilaymiz
			if !isCurrentlyUp && len(allRecentOutages) > 0 {
				last := &allRecentOutages[len(allRecentOutages)-1]
				if last.ServiceName == m.Name {
					last.IsOngoing = true
				}
			}

			compName := strings.ToUpper(m.ComponentType[:1]) + m.ComponentType[1:]

			projectList[projIdx].data.Components = append(projectList[projIdx].data.Components, ComponentData{
				Name:              compName,
				Type:              m.ComponentType,
				IsUp:              isCurrentlyUp,
				UptimePct:         uptimePct,
				Latency:           latency,
				TotalDowntimeSecs: totalDowntime,
				TotalOutages:      len(rawOutages),
				LongestOutageSecs: longestOutage,
				CreatedAt:         m.CreatedAt.Format(time.RFC3339),
				History:           history,
			})
		}

		// 3. Response
		overallUptime := 100.0
		if totalUptimeCount > 0 {
			overallUptime = totalUptimeSum / float64(totalUptimeCount)
		}

		// Maintenance Logs
		maintenanceLogs, _ := db.GetRecentMaintenanceLogs(10)
		if maintenanceLogs == nil {
			maintenanceLogs = []models.MaintenanceLog{}
		}

		// RecentOutages ni yangilari birinchi bo'lishi uchun tartiblash
		sortRecentOutages(allRecentOutages)

		// Faqat so'nggi 20 ta outage
		if len(allRecentOutages) > 20 {
			allRecentOutages = allRecentOutages[:20]
		}
		if allRecentOutages == nil {
			allRecentOutages = []RecentOutage{}
		}

		response := PublicStatusResponse{
			SiteName:        siteName,
			LastUpdated:     time.Now().UTC().Format(time.RFC3339),
			OverallUptime:   overallUptime,
			TotalServices:   totalUptimeCount,
			ActiveIncidents: activeIncidents,
			Projects:        []PublicProjectData{},
			RecentOutages:   allRecentOutages,
			MaintenanceLogs: maintenanceLogs,
		}

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
}

// flatOutage — buildHistory'dan chiqadigan outage
type flatOutage struct {
	Start           string
	End             string
	DurationSeconds int
}

func sortRecentOutages(outages []RecentOutage) {
	// Yangi start birinchi (teskari saralash)
	for i := 0; i < len(outages)-1; i++ {
		for j := 0; j < len(outages)-1-i; j++ {
			if outages[j].Start < outages[j+1].Start {
				outages[j], outages[j+1] = outages[j+1], outages[j]
			}
		}
	}
}

func buildHistory(m models.Monitor, days int) ([]DailyStatus, int, int, []flatOutage) {
	var history []DailyStatus
	now := time.Now().UTC()

	daysMap := make(map[string]*DailyStatus)
	for i := days - 1; i >= 0; i-- {
		dateStr := now.AddDate(0, 0, -i).Format("2006-01-02")
		d := DailyStatus{Date: dateStr, IsUp: true, Pct: 100.0, HasData: true, Outages: []Outage{}}
		history = append(history, d)
		daysMap[dateStr] = &history[len(history)-1]
	}

	startDateStr := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")

	rows, err := db.DB.Query(`
		SELECT is_up, checked_at 
		FROM heartbeats 
		WHERE monitor_id = ? AND checked_at >= ?
		ORDER BY checked_at ASC
	`, m.ID, startDateStr+" 00:00:00")

	var hbs []struct {
		IsUp      bool
		CheckedAt time.Time
	}

	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var isUp bool
			var checkedAtStr string
			if err := rows.Scan(&isUp, &checkedAtStr); err == nil {
				if t, err := time.Parse("2006-01-02 15:04:05", checkedAtStr); err == nil {
					hbs = append(hbs, struct {
						IsUp      bool
						CheckedAt time.Time
					}{IsUp: isUp, CheckedAt: t})
				} else if t, err := time.Parse(time.RFC3339, checkedAtStr); err == nil {
					hbs = append(hbs, struct {
						IsUp      bool
						CheckedAt time.Time
					}{IsUp: isUp, CheckedAt: t})
				}
			}
		}
		if err := rows.Err(); err != nil {
			// Iteration xatosi — to'plangan hbs bilan davom etamiz
			_ = err
		}
	}

	type rawOutage struct {
		start time.Time
		end   time.Time
	}
	var rawOutages []rawOutage

	var currentOutageStart *time.Time
	var lastCheckedAt time.Time

	// 2.5x grace period — monitor restart bo'lsa false positive bo'lmasin
	gracePeriod := time.Duration(float64(m.IntervalSeconds)*2.5) * time.Second

	for _, hb := range hbs {
		if !lastCheckedAt.IsZero() {
			gap := hb.CheckedAt.Sub(lastCheckedAt)
			if gap > gracePeriod {
				outageStart := lastCheckedAt.Add(time.Duration(m.IntervalSeconds) * time.Second)
				if currentOutageStart == nil {
					t := outageStart
					currentOutageStart = &t
				}
			}
		}

		if !hb.IsUp {
			if currentOutageStart == nil {
				t := hb.CheckedAt
				currentOutageStart = &t
			}
		} else {
			if currentOutageStart != nil {
				rawOutages = append(rawOutages, rawOutage{
					start: *currentOutageStart,
					end:   hb.CheckedAt,
				})
				currentOutageStart = nil
			}
		}
		lastCheckedAt = hb.CheckedAt
	}

	if currentOutageStart != nil {
		rawOutages = append(rawOutages, rawOutage{
			start: *currentOutageStart,
			end:   now,
		})
	}

	var allFlatOutages []flatOutage

	for _, out := range rawOutages {
		totalDur := int(out.end.Sub(out.start).Seconds())
		if totalDur <= 0 {
			continue
		}
		allFlatOutages = append(allFlatOutages, flatOutage{
			Start:           out.start.Format(time.RFC3339),
			End:             out.end.Format(time.RFC3339),
			DurationSeconds: totalDur,
		})

		// Kunlik taqsimlash
		currStart := out.start
		for currStart.Before(out.end) {
			dateStr := currStart.Format("2006-01-02")
			endOfDay := time.Date(currStart.Year(), currStart.Month(), currStart.Day(), 23, 59, 59, 999999999, currStart.Location())

			currEnd := out.end
			if currEnd.After(endOfDay) {
				currEnd = endOfDay
			}

			if day, ok := daysMap[dateStr]; ok {
				dur := int(currEnd.Sub(currStart).Seconds())
				if dur > 0 {
					day.Outages = append(day.Outages, Outage{
						Start:           currStart.Format(time.RFC3339),
						End:             currEnd.Format(time.RFC3339),
						DurationSeconds: dur,
					})
					day.IsUp = false
				}
			}

			currStart = endOfDay.Add(1 * time.Nanosecond)
		}
	}

	// Uptime % hisoblash
	totalSeconds := 0
	downtimeSeconds := 0

	for i := range history {
		day := &history[i]

		// If the end of this day is before monitor was created, it has no data
		dayEndTime, _ := time.Parse("2006-01-02", day.Date)
		dayEndTime = dayEndTime.Add(24 * time.Hour).Add(-1 * time.Nanosecond)
		if dayEndTime.Before(m.CreatedAt) {
			day.HasData = false
			day.Pct = 0
			continue
		}

		totalDowntime := 0
		for _, o := range day.Outages {
			totalDowntime += o.DurationSeconds
		}

		var totalSecondsInDay int
		if day.Date == now.Format("2006-01-02") {
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			totalSecondsInDay = int(now.Sub(startOfDay).Seconds())
		} else {
			totalSecondsInDay = 86400
		}

		if totalSecondsInDay <= 0 {
			day.Pct = 100.0
		} else {
			uptime := totalSecondsInDay - totalDowntime
			if uptime < 0 {
				uptime = 0
			}
			day.Pct = (float64(uptime) / float64(totalSecondsInDay)) * 100.0
		}

		totalSeconds += totalSecondsInDay
		downtimeSeconds += totalDowntime
	}

	return history, totalSeconds, downtimeSeconds, allFlatOutages
}
