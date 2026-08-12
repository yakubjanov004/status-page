package discovery

import (
	"log"
	"strings"

	"status-page/internal/db"
)

// AutoSeed - agar DB bo'sh bo'lsa, Nginx va Systemd fayllarini skanerlab
// loyihalar va monitorlarni avtomatik qo'shadi. Idempotent (bir necha marta
// ishlatsa ham xavfsiz - faqat projects = 0 bo'lganda ishlaydi).
func AutoSeed() {
	var count int
	err := db.DB.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&count)
	if err != nil {
		log.Printf("[SEED] Could not check project count: %v", err)
		return
	}
	if count > 0 {
		log.Printf("[SEED] %d projects already exist. Skipping auto-seed.", count)
		return
	}

	log.Println("[SEED] DB is empty. Starting auto-discovery seed...")

	items := DiscoverAll()
	if len(items) == 0 {
		log.Println("[SEED] No items discovered. Make sure DISCOVERY_NGINX_DIR and DISCOVERY_SYSTEMD_DIR are set in .env")
		return
	}
	log.Printf("[SEED] Discovered %d items total.", len(items))

	// Loyihalar bo'yicha guruhlash (kichik harfda nomi key)
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

	// Har bir loyiha uchun DB ga yozamiz
	for _, key := range order {
		g := groups[key]
		slug := strings.ToLower(
			strings.ReplaceAll(
				strings.ReplaceAll(g.displayName, " ", "-"),
				"_", "-",
			),
		)

		// Projects jadvaliga qo'shamiz
		res, err := db.DB.Exec(
			`INSERT OR IGNORE INTO projects (name, slug, description, show_on_public_page) VALUES (?, ?, ?, ?)`,
			g.displayName, slug, "", true,
		)
		if err != nil {
			log.Printf("[SEED] Failed to create project '%s': %v", g.displayName, err)
			continue
		}
		projectID, _ := res.LastInsertId()
		log.Printf("[SEED] Created project: '%s' (id=%d)", g.displayName, projectID)

		// Har bir discovered item uchun monitor yaratamiz
		seenURLs := make(map[string]bool)
		for _, item := range g.items {
			if item.URL == "" {
				log.Printf("[SEED]   SKIP '%s' — URL topilmadi", item.RawName)
				continue
			}
			if seenURLs[item.URL] {
				log.Printf("[SEED]   SKIP '%s' — URL takroriy (%s)", item.RawName, item.URL)
				continue
			}
			seenURLs[item.URL] = true

			componentType := item.Component
			if componentType == "unknown" || componentType == "" {
				componentType = "backend"
			}

			monitorName := g.displayName + " " + strings.Title(componentType)
			_, err := db.DB.Exec(
				`INSERT INTO monitors (name, type, url, interval_seconds, timeout_seconds, expected_status_code, is_active, show_on_public_page, project_id, component_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				monitorName,
				"http",
				item.URL,
				60,   // 60 soniyada bir tekshiriladi
				10,   // 10 soniya timeout
				200,  // 200 kutiladi
				true, // aktiv
				true, // public sahifada ko'rinadi
				int(projectID),
				componentType,
			)
			if err != nil {
				log.Printf("[SEED]   FAILED monitor '%s' -> %s: %v", item.RawName, item.URL, err)
			} else {
				log.Printf("[SEED]   OK monitor: '%s' [%s] -> %s", monitorName, componentType, item.URL)
			}
		}
	}

	log.Println("[SEED] Auto-seed completed successfully!")
}
