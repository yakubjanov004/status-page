package handler

import (
	"status-page/internal/webhookdb"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRoutes wires all webhook API routes onto the given chi mux.
func SetupRoutes(r *chi.Mux, db *webhookdb.DB, hookToken string) {
	h := New(db, hookToken)

	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Health check (no auth)
	r.Get("/healthz", h.HandleHealthz)

	// Status page (no auth)
	r.Get("/status", h.HandleStatusPage)

	// Public read-only API (no auth required for read endpoints)
	r.Route("/api/v1", func(api chi.Router) {
		// Webhook endpoint requires token auth
		api.With(h.requireToken).Post("/webhook", h.HandleWebhook)

		// Read-only endpoints
		api.Get("/services", h.HandleListServices)
		api.Get("/services/{name}/incidents", h.HandleListIncidents)
		api.Get("/services/{name}/uptime", h.HandleUptime)
	})
}
