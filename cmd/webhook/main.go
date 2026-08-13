package main

import (
	"log"
	"net/http"
	"os"

	"status-page/internal/handler"
	"status-page/internal/webhookdb"

	"github.com/go-chi/chi/v5"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	hookToken := os.Getenv("HOOK_TOKEN")
	if hookToken == "" {
		log.Fatal("[webhook] HOOK_TOKEN env var is required")
	}

	// Optional: HMAC_SECRET enables X-Hub-Signature-256 verification.
	// If not set, token-only auth is used (backward compatible).
	hmacSecret := os.Getenv("HMAC_SECRET")
	if hmacSecret != "" {
		log.Printf("[webhook] HMAC-SHA256 signature verification enabled")
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "./data/status.db"
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	// Initialize database (runs migrations + seeds services)
	db, err := webhookdb.Init(dbPath)
	if err != nil {
		log.Fatalf("[webhook] failed to initialize database: %v", err)
	}
	defer db.Close()

	// Setup HTTP routes
	r := chi.NewRouter()
	handler.SetupRoutes(r, db, hookToken, hmacSecret)

	log.Printf("[webhook] starting server on %s (db=%s)", listenAddr, dbPath)
	if err := http.ListenAndServe(listenAddr, r); err != nil {
		log.Fatalf("[webhook] server failed: %v", err)
	}
}
