package api

import (
	"net/http"
	"status-page/internal/config"
	"status-page/internal/websocket"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
	}))

	// Public API — cfg closure orqali handler'ga uzatiladi
	r.Get("/api/public/status", GetPublicStatusHandler(cfg))

	// WebSocket
	r.Get("/ws", websocket.GlobalHub.HandleWebSocket)

	// Admin — redirect to home
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	})

	return r
}

// SpaHandler — public sahifani serve qiladi, cache-busting bilan
func SpaHandler(publicDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		// Cache-busting headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, publicDir+"/public/index.html")
	}
}
