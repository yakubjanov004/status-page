package api

import (
	"encoding/json"
	"net/http"
	"status-page/internal/config"
	"status-page/internal/db"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type ProjectDetailResponse struct {
	ID            int             `json:"id"`
	Name          string          `json:"name"`
	Slug          string          `json:"slug"`
	Components    []ComponentData `json:"components"`
	Incidents     []RecentOutage  `json:"incidents"`
	RestartCount  int             `json:"restart_count"`
	LastRestartAt string          `json:"last_restart_at,omitempty"`
}

func GetProjectStatusHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		slug := chi.URLParam(r, "slug")
		if slug == "" {
			http.Error(w, "slug required", http.StatusBadRequest)
			return
		}

		rangeStr := r.URL.Query().Get("range")
		days := 7
		switch rangeStr {
		case "30", "30d":
			days = 30
		case "90", "90d":
			days = 90
		}

		var projectID int
		var projectName string
		err := db.DB.QueryRow(`SELECT id, name FROM projects WHERE slug = ?`, slug).Scan(&projectID, &projectName)
		if err != nil {
			http.Error(w, "project not found", http.StatusNotFound)
			return
		}

		monitors, err := db.GetAllMonitors()
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		restartStats, err := db.GetRestartStats()
		if err != nil {
			restartStats = map[string]db.RestartStat{}
		}

		var components []ComponentData
		var allRecentOutages []RecentOutage
		totalRestarts := 0
		var lastRestartAt time.Time

		for _, m := range monitors {
			if m.ProjectID == nil || *m.ProjectID != projectID || !m.ShowOnPublicPage {
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

			history, totalSeconds, downtimeSeconds, rawOutages := buildHistory(m, days)

			uptimePct := 100.0
			if totalSeconds > 0 {
				uptime := totalSeconds - downtimeSeconds
				if uptime < 0 {
					uptime = 0
				}
				uptimePct = float64(uptime) / float64(totalSeconds) * 100.0
			}

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

			if !isCurrentlyUp && len(allRecentOutages) > 0 {
				last := &allRecentOutages[len(allRecentOutages)-1]
				if last.ServiceName == m.Name {
					last.IsOngoing = true
				}
			}

			compName := strings.ToUpper(m.ComponentType[:1]) + m.ComponentType[1:]

			restartCount := 0
			compLastRestartAt := ""
			restartKey := db.NormalizeServiceKey(slug + "-" + m.ComponentType)
			if st, ok := restartStats[restartKey]; ok {
				restartCount = st.Count
				compLastRestartAt = st.LastAt.UTC().Format(time.RFC3339)
				totalRestarts += st.Count
				if st.LastAt.After(lastRestartAt) {
					lastRestartAt = st.LastAt
				}
			}

			components = append(components, ComponentData{
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
				RestartCount:      restartCount,
				LastRestartAt:     compLastRestartAt,
			})
		}

		sortRecentOutages(allRecentOutages)
		if allRecentOutages == nil {
			allRecentOutages = []RecentOutage{}
		}

		response := ProjectDetailResponse{
			ID:           projectID,
			Name:         projectName,
			Slug:         slug,
			Components:   components,
			Incidents:    allRecentOutages,
			RestartCount: totalRestarts,
		}
		if !lastRestartAt.IsZero() {
			response.LastRestartAt = lastRestartAt.Format(time.RFC3339)
		}

		json.NewEncoder(w).Encode(response)
	}
}
