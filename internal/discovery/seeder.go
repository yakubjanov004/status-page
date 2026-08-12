package discovery

import (
	"log"

	"status-page/internal/db"
)

type projectDef struct {
	Name       string
	Slug       string
	Components []componentDef
}

type componentDef struct {
	Name          string
	ComponentType string
	URL           string
}

// To'g'ri portlar — serverdan olingan
var projects = []projectDef{
	{
		Name: "Datan",
		Slug: "datan",
		Components: []componentDef{
			{Name: "Datan Backend", ComponentType: "backend", URL: "http://127.0.0.1:8003"},
		},
	},
	{
		Name: "Tokpoint",
		Slug: "tokpoint",
		Components: []componentDef{
			{Name: "Tokpoint Frontend", ComponentType: "frontend", URL: "http://127.0.0.1:5173"},
			{Name: "Tokpoint Backend", ComponentType: "backend", URL: "http://127.0.0.1:8001"},
		},
	},
	{
		Name: "Odimrepo",
		Slug: "odimrepo",
		Components: []componentDef{
			{Name: "Odimrepo Frontend", ComponentType: "frontend", URL: "http://127.0.0.1:5120"},
			{Name: "Odimrepo Backend", ComponentType: "backend", URL: "http://127.0.0.1:8010"},
		},
	},
	{
		Name: "Mehmonxona",
		Slug: "mehmonxona",
		Components: []componentDef{
			{Name: "Mehmonxona App", ComponentType: "backend", URL: "http://127.0.0.1:3000"},
		},
	},
}

func AutoSeed() {
	log.Println("[SEED] Syncing hardcoded projects...")

	for _, p := range projects {
		var projectID int64
		err := db.DB.QueryRow(`SELECT id FROM projects WHERE slug = ?`, p.Slug).Scan(&projectID)
		if err != nil {
			res, err := db.DB.Exec(
				`INSERT INTO projects (name, slug, description, show_on_public_page) VALUES (?, ?, ?, ?)`,
				p.Name, p.Slug, "", true,
			)
			if err != nil {
				log.Printf("[SEED] Failed to create project '%s': %v", p.Name, err)
				continue
			}
			projectID, _ = res.LastInsertId()
			log.Printf("[SEED] Created project: '%s' (id=%d)", p.Name, projectID)
		} else {
			log.Printf("[SEED] Project '%s' exists (id=%d)", p.Name, projectID)
		}

		for _, c := range p.Components {
			var existing int
			db.DB.QueryRow(
				`SELECT COUNT(*) FROM monitors WHERE url = ? AND project_id = ?`,
				c.URL, projectID,
			).Scan(&existing)

			if existing > 0 {
				continue
			}

			_, err := db.DB.Exec(
				`INSERT INTO monitors (name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				c.Name, "http", c.URL,
				60, 10, 200, true, true,
				int(projectID), c.ComponentType,
			)
			if err != nil {
				log.Printf("[SEED]   FAILED: %s -> %s: %v", c.Name, c.URL, err)
			} else {
				log.Printf("[SEED]   OK: %s [%s] -> %s", c.Name, c.ComponentType, c.URL)
			}
		}
	}

	log.Println("[SEED] Done.")
}
