package api

import (
	"encoding/json"
	"net/http"
	"status-page/internal/db"
	"status-page/internal/models"
)

func GetProjectsHandler(w http.ResponseWriter, r *http.Request) {
	// For simplicity, we just fetch all projects and their monitors, 
	// then group them manually.
	rows, err := db.DB.Query(`SELECT id, name, slug, description, show_on_public_page, created_at FROM projects`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.ShowOnPublicPage, &p.CreatedAt); err != nil {
			continue
		}
		projects = append(projects, p)
	}

	monitors, _ := db.GetAllMonitors()
	
	for i := range projects {
		for _, m := range monitors {
			if m.ProjectID != nil && *m.ProjectID == projects[i].ID {
				mCopy := m
				if m.ComponentType == "frontend" {
					projects[i].FrontendMonitor = &mCopy
				} else if m.ComponentType == "backend" {
					projects[i].BackendMonitor = &mCopy
				}
			}
		}
	}

	if projects == nil {
		projects = []models.Project{}
	}
	json.NewEncoder(w).Encode(projects)
}

func CreateProjectHandler(w http.ResponseWriter, r *http.Request) {
	var p models.Project
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err := db.DB.QueryRow(`INSERT INTO projects (name, slug, description, show_on_public_page) VALUES (?, ?, ?, ?) RETURNING id`,
		p.Name, p.Slug, p.Description, p.ShowOnPublicPage).Scan(&p.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}
