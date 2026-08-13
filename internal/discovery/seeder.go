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

// Faqat 5 ta loyiha — serverda haqiqatan ishlaydigan va keraklilari
// Tokpoint: Docker orqali run qilingan, backend 8001 portda
// Nginx tokpoint.darrov.uz uchun frontend'ni statik beradi (separate monitor kerak emas)
var projects = []projectDef{
	{
		// tokpoint-docker.service: docker compose up -> backend 8001 portni expose qiladi
		// nginx: /api/ -> 127.0.0.1:8001, / -> static /var/www/tokpoint/frontend/dist
		Name: "Tokpoint",
		Slug: "tokpoint",
		Components: []componentDef{
			{Name: "Tokpoint Backend", ComponentType: "backend", URL: "http://127.0.0.1:8001"},
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
		// datan.service: gunicorn --bind 127.0.0.1:8003
		Name: "Datan",
		Slug: "datan",
		Components: []componentDef{
			{Name: "Datan Backend", ComponentType: "backend", URL: "http://127.0.0.1:8003"},
		},
	},
	{
		// alfaconnect-webapp.service: npm run start -> PORT=5175
		// alfaconnect-bot.service: FastAPI -> 8002 (nginx /api/ -> 8002)
		Name: "AlfaConnect",
		Slug: "alfaconnect",
		Components: []componentDef{
			{Name: "AlfaConnect WebApp", ComponentType: "frontend", URL: "http://127.0.0.1:5175"},
			{Name: "AlfaConnect API", ComponentType: "backend", URL: "http://127.0.0.1:8002"},
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
}

// activeslugs — public page'da ko'rsatiladigan loyihalar
var activeSlugs = []string{"tokpoint", "odimrepo", "datan", "alfaconnect", "mehmonxona"}

func AutoSeed() {
	log.Println("[SEED] Syncing projects...")

	// Birinchi: barcha loyihalarni yashiramiz, keyin faqat keraklisini yoqamiz
	hideUnused()

	for _, p := range projects {
		var projectID int64
		err := db.DB.QueryRow(`SELECT id FROM projects WHERE slug = ?`, p.Slug).Scan(&projectID)
		if err != nil {
			// Yangi loyiha yaratish
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
			// Mavjud loyihani ko'rinishini yoqish
			db.DB.Exec(`UPDATE projects SET show_on_public_page = 1 WHERE id = ?`, projectID)
			log.Printf("[SEED] Project '%s' exists (id=%d)", p.Name, projectID)
		}

		for _, c := range p.Components {
			var monitorID int
			err := db.DB.QueryRow(
				`SELECT id FROM monitors WHERE url = ? AND project_id = ?`,
				c.URL, projectID,
			).Scan(&monitorID)

			if err != nil {
				// Yangi monitor qo'shish
				// expected_status_code = 0 => 200-399 oraliq barchasi UP
				_, err := db.DB.Exec(
					`INSERT INTO monitors (name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type)
					 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					c.Name, "http", c.URL,
					60, 10, 0, true, true,
					int(projectID), c.ComponentType,
				)
				if err != nil {
					log.Printf("[SEED]   FAILED: %s -> %s: %v", c.Name, c.URL, err)
				} else {
					log.Printf("[SEED]   OK (new): %s [%s] -> %s", c.Name, c.ComponentType, c.URL)
				}
			} else {
				// Mavjud monitorni yangilash — expected_status_code va show_on_public_page to'g'rilash
				db.DB.Exec(
					`UPDATE monitors SET expected_status_code = 0, show_on_public_page = 1, is_active = 1, name = ?, component_type = ? WHERE id = ?`,
					c.Name, c.ComponentType, monitorID,
				)
				log.Printf("[SEED]   OK (updated): %s [%s] -> %s", c.Name, c.ComponentType, c.URL)
			}
		}
	}

	log.Println("[SEED] Done.")
}

// hideUnused — ro'yxatda bo'lmagan loyihalarni yashiradi
func hideUnused() {
	if len(activeSlugs) == 0 {
		return
	}
	// Querry placeholder'larini yasaymiz
	placeholders := ""
	args := make([]interface{}, len(activeSlugs))
	for i, s := range activeSlugs {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args[i] = s
	}
	db.DB.Exec(
		`UPDATE projects SET show_on_public_page = 0 WHERE slug NOT IN (`+placeholders+`)`,
		args...,
	)
	log.Println("[SEED] Hidden unused projects from public page")
}
