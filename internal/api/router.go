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

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
	}))

	// Faqat public API — admin panel yo'q
	r.Get("/api/public/status", GetPublicStatusHandler)

	// WebSocket — real-time yangilanishlar
	r.Get("/ws", websocket.GlobalHub.HandleWebSocket)

	return r
}

// SpaHandler — faqat public status sahifasini serve qiladi
func SpaHandler(publicDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, publicDir+"/public/index.html")
	}
}
