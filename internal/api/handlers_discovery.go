package api

import (
	"encoding/json"
	"net/http"
	"status-page/internal/db"
	"status-page/internal/discovery"
	"status-page/internal/models"
	"status-page/internal/monitor"
)

func ScanDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	results := discovery.DiscoverAll()
	if results == nil {
		results = []discovery.DiscoveredItem{}
	}
	json.NewEncoder(w).Encode(results)
}

type AddDiscoveredRequest struct {
	ProjectID int    `json:"project_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`      // 'http' or 'tcp'
	URL       string `json:"url"`
	Component string `json:"component"` // 'frontend' or 'backend'
}

func AddDiscoveredProjectHandler(w http.ResponseWriter, r *http.Request) {
	var req AddDiscoveredRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Make sure project exists
	var p models.Project
	err := db.DB.QueryRow(`SELECT id, name FROM projects WHERE id = ?`, req.ProjectID).Scan(&p.ID, &p.Name)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	m := models.Monitor{
		Name:               req.Name,
		Type:               req.Type,
		URL:                req.URL,
		IntervalSeconds:    60,
		TimeoutSeconds:     10,
		ExpectedStatusCode: 200,
		IsActive:           true,
		ShowOnPublicPage:   true,
		ProjectID:          &req.ProjectID,
		ComponentType:      req.Component,
	}

	if err := db.CreateMonitor(&m); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	monitor.GlobalScheduler.StartMonitor(&m)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
