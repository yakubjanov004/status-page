package handler

import (
	"status-page/internal/webhookdb"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRoutes wires all webhook API routes onto the given chi mux.
func SetupRoutes(r *chi.Mux, db *webhookdb.DB, hookToken, hmacSecret string) {
	h := NewWithHMAC(db, hookToken, hmacSecret)

	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(requestIDMiddleware) // injects/echoes X-Request-Id on every response

	// Health check (no auth)
	r.Get("/healthz", h.HandleHealthz)

	// Status page (no auth)
	r.Get("/status", h.HandleStatusPage)

	// Public read-only API (no auth required for read endpoints)
	r.Route("/api/v1", func(api chi.Router) {
		// Webhook endpoint: token auth + optional HMAC signature verification
		api.With(h.requireToken, h.verifyHMAC).Post("/webhook", h.HandleWebhook)

		// Read-only endpoints
		api.Get("/services", h.HandleListServices)
		api.Get("/services/{name}/incidents", h.HandleListIncidents)
		api.Get("/services/{name}/uptime", h.HandleUptime)
	})
}

