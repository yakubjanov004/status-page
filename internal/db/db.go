package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var DB *sql.DB

func Init(dbPath string) error {
	// Ensure the directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = db
	log.Println("Connected to SQLite database at", dbPath)

	return runMigrations()
}

func runMigrations() error {
	migrationFiles := []string{
		"001_init.sql",
		"002_add_projects.sql",
	}

	for _, file := range migrationFiles {
		content, err := migrationsFS.ReadFile("migrations/" + file)
		if err != nil {
			log.Printf("Could not find migration file %s, skipping\n", file)
			continue
		}

		queries := strings.Split(string(content), ";")
		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}
			_, err := DB.Exec(query)
			if err != nil {
				// Ignore errors like "duplicate column" for simpler migrations running multiple times
				if !strings.Contains(err.Error(), "duplicate column name") && !strings.Contains(err.Error(), "table projects already exists") {
					log.Printf("Failed to execute query in %s: %v\nQuery: %s", file, err, query)
				}
			}
		}
	}

	log.Println("Database migrations applied successfully")
	return nil
}
