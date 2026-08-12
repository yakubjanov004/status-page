package api

import (
	"encoding/json"
	"net/http"
	"status-page/internal/db"
	"status-page/internal/models"
	"status-page/internal/monitor"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func GetMonitorsHandler(w http.ResponseWriter, r *http.Request) {
	monitors, err := db.GetAllMonitors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if monitors == nil {
		monitors = []models.Monitor{}
	}
	json.NewEncoder(w).Encode(monitors)
}

func CreateMonitorHandler(w http.ResponseWriter, r *http.Request) {
	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := db.CreateMonitor(&m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Start scheduling for this new monitor
	if m.IsActive {
		monitor.GlobalScheduler.StartMonitor(&m)
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(m)
}

func UpdateMonitorHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	var m models.Monitor
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	m.ID = id

	if err := db.UpdateMonitor(&m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update scheduling
	if m.IsActive {
		monitor.GlobalScheduler.StartMonitor(&m)
	} else {
		monitor.GlobalScheduler.StopMonitor(m.ID)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(m)
}

func DeleteMonitorHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	if err := db.DeleteMonitor(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	monitor.GlobalScheduler.StopMonitor(id)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func GetMonitorHistoryHandler(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid monitor ID", http.StatusBadRequest)
		return
	}

	hbs, err := db.GetRecentHeartbeats(id, 90)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if hbs == nil {
		hbs = []models.Heartbeat{}
	}
	json.NewEncoder(w).Encode(hbs)
}
