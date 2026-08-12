package discovery

import (
	"log"
	"strings"

	"status-page/internal/db"
)

// AutoSeed — har safar dastur ishga tushganda Nginx va Systemd fayllarini
// skanerlab, yangi topilgan loyiha/monitorlarni DB ga qo'shadi.
// Idempotent: mavjudlarni qayta yaratmaydi.
func AutoSeed() {
	log.Println("[SEED] Syncing projects from Nginx/Systemd configs...")

	items := DiscoverAll()
	if len(items) == 0 {
		log.Println("[SEED] No items discovered. Check DISCOVERY_NGINX_DIR and DISCOVERY_SYSTEMD_DIR in .env")
		return
	}
	log.Printf("[SEED] Discovered %d items.", len(items))

	// Loyihalar bo'yicha guruhlash
	type group struct {
		displayName string
		items       []DiscoveredItem
	}
	groups := make(map[string]*group)
	var order []string

	for _, item := range items {
		key := strings.ToLower(item.Project)
		if _, exists := groups[key]; !exists {
			groups[key] = &group{displayName: item.Project}
			order = append(order, key)
		}
		groups[key].items = append(groups[key].items, item)
	}

	for _, key := range order {
		g := groups[key]
		slug := strings.ToLower(
			strings.NewReplacer(" ", "-", "_", "-").Replace(g.displayName),
		)

		// Loyiha mavjudmi? Yo'q bo'lsa yaratamiz
		var projectID int64
		err := db.DB.QueryRow(`SELECT id FROM projects WHERE slug = ?`, slug).Scan(&projectID)
		if err != nil {
			// Loyiha yo'q — yaratamiz
			res, err := db.DB.Exec(
				`INSERT INTO projects (name, slug, description, show_on_public_page) VALUES (?, ?, ?, ?)`,
				g.displayName, slug, "", true,
			)
			if err != nil {
				log.Printf("[SEED] Failed to create project '%s': %v", g.displayName, err)
				continue
			}
			projectID, _ = res.LastInsertId()
			log.Printf("[SEED] Created project: '%s' (id=%d)", g.displayName, projectID)
		} else {
			log.Printf("[SEED] Project '%s' already exists (id=%d)", g.displayName, projectID)
		}

		// Har bir item uchun monitor — URL takrorlanmaydi
		seenURLs := make(map[string]bool)
		for _, item := range g.items {
			if item.URL == "" {
				continue
			}
			if seenURLs[item.URL] {
				continue
			}
			seenURLs[item.URL] = true

			componentType := item.Component
			if componentType == "unknown" || componentType == "" {
				componentType = "backend"
			}

			// Monitor allaqachon bormi?
			var existing int
			db.DB.QueryRow(
				`SELECT COUNT(*) FROM monitors WHERE url = ? AND project_id = ?`,
				item.URL, projectID,
			).Scan(&existing)

			if existing > 0 {
				log.Printf("[SEED]   SKIP monitor (exists): %s -> %s", componentType, item.URL)
				continue
			}

			monitorName := g.displayName + " " + strings.Title(componentType)
			_, err := db.DB.Exec(
				`INSERT INTO monitors (name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				monitorName, "http", item.URL,
				60, 10, 200, true, true,
				int(projectID), componentType,
			)
			if err != nil {
				log.Printf("[SEED]   FAILED monitor '%s' -> %s: %v", item.RawName, item.URL, err)
			} else {
				log.Printf("[SEED]   OK monitor: '%s' [%s] -> %s", monitorName, componentType, item.URL)
			}
		}
	}

	log.Println("[SEED] Sync completed.")
}
