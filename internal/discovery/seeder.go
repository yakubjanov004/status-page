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

// Serverdan olingan haqiqiy portlar — nginx va systemd fayllariga asoslanib
var projects = []projectDef{
	{
		// tokpoint-frontend.service: serve -s dist -l 5173
		// tokpoint-backend.service: uvicorn --port 8001
		Name: "Tokpoint",
		Slug: "tokpoint",
		Components: []componentDef{
			{Name: "Tokpoint Frontend", ComponentType: "frontend", URL: "http://127.0.0.1:5173"},
			{Name: "Tokpoint Backend", ComponentType: "backend", URL: "http://127.0.0.1:8001"},
		},
	},
	{
		// dsktp3-frontend.service: serve -s dist -l 5177
		// dsktp3-backend-bot.service: python run.py -> 8100
		// nginx dsktp3.conf: darrov.uz / -> 5177, /api/ -> 8100
		Name: "DSKTP3",
		Slug: "dsktp3",
		Components: []componentDef{
			{Name: "DSKTP3 Frontend", ComponentType: "frontend", URL: "http://127.0.0.1:5177"},
			{Name: "DSKTP3 Backend", ComponentType: "backend", URL: "http://127.0.0.1:8100"},
		},
	},
	{
		// alfaconnect-webapp.service: npm run start -> PORT=5175
		// alfaconnect-bot.service: python main.py -> 8002 (nginx alfaconnect.conf /api/ -> 8002)
		Name: "AlfaConnect",
		Slug: "alfaconnect",
		Components: []componentDef{
			{Name: "AlfaConnect WebApp", ComponentType: "frontend", URL: "http://127.0.0.1:5175"},
			{Name: "AlfaConnect API", ComponentType: "backend", URL: "http://127.0.0.1:8002"},
		},
	},
	{
		// odimrepo-frontend.service: serve -s dist -l 5120
		// odimrepo-backend.service: daphne -p 8010
		Name: "Odimrepo",
		Slug: "odimrepo",
		Components: []componentDef{
			{Name: "Odimrepo Frontend", ComponentType: "frontend", URL: "http://127.0.0.1:5120"},
			{Name: "Odimrepo Backend", ComponentType: "backend", URL: "http://127.0.0.1:8010"},
		},
	},
	{
		// memories-frontend.service: npm run preview --port 5163
		// memories.service: uvicorn --port 8013
		Name: "Memories",
		Slug: "memories",
		Components: []componentDef{
			{Name: "Memories Frontend", ComponentType: "frontend", URL: "http://127.0.0.1:5163"},
			{Name: "Memories Backend", ComponentType: "backend", URL: "http://127.0.0.1:8013"},
		},
	},
	{
		// mehmonxona.service: node backend/server.js -> 3000
		Name: "Mehmonxona",
		Slug: "mehmonxona",
		Components: []componentDef{
			{Name: "Mehmonxona App", ComponentType: "backend", URL: "http://127.0.0.1:3000"},
		},
	},
	{
		// chorva.service: uvicorn --port 8004
		// nginx apichorva.conf: apichorva.yoqubjonov.me -> 8004
		Name: "Chorva",
		Slug: "chorva",
		Components: []componentDef{
			{Name: "Chorva API", ComponentType: "backend", URL: "http://127.0.0.1:8004"},
		},
	},
	{
		// investconnect.service: gunicorn --bind 127.0.0.1:8005
		Name: "InvestConnect",
		Slug: "investconnect",
		Components: []componentDef{
			{Name: "InvestConnect API", ComponentType: "backend", URL: "http://127.0.0.1:8005"},
		},
	},
	{
		// battle-royale.service: node server.js -> PORT=5330
		// nginx jang.games.conf: jang.games -> 5330
		Name: "Battle Royale",
		Slug: "battle-royale",
		Components: []componentDef{
			{Name: "Battle Royale Server", ComponentType: "backend", URL: "http://127.0.0.1:5330"},
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
				60, 10, 0, true, true,
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
