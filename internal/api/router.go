package api

import (
	"status-page/internal/config"
	"status-page/internal/websocket"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // Adjust in prod
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
	}))

	r.Route("/api", func(r chi.Router) {
		r.Post("/auth/login", LoginHandler(cfg))
		r.Post("/auth/logout", LogoutHandler())
		r.Get("/auth/check", CheckAuthHandler(cfg))

		r.Get("/public/status", GetPublicStatusHandler)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(cfg))
			
			r.Get("/monitors", GetMonitorsHandler)
			r.Post("/monitors", CreateMonitorHandler)
			r.Put("/monitors/{id}", UpdateMonitorHandler)
			r.Delete("/monitors/{id}", DeleteMonitorHandler)
			r.Get("/monitors/{id}/history", GetMonitorHistoryHandler)
			
			r.Get("/projects", GetProjectsHandler)
			r.Post("/projects", CreateProjectHandler)
			
			r.Get("/discovery/scan", ScanDiscoveryHandler)
			r.Post("/discovery/add", AddDiscoveredProjectHandler)
		})
	})

	r.Get("/ws", websocket.GlobalHub.HandleWebSocket)

	return r
}
