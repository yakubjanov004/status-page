package api

import (
	"net/http"
	"os"
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
		AllowedMethods: []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Content-Type"},
	}))

	// Public API — cfg closure orqali handler'ga uzatiladi
	r.Get("/api/public/status", GetPublicStatusHandler(cfg))
	r.Get("/api/public/status/project/{slug}", GetProjectStatusHandler(cfg))

	// Internal API — systemd service-notify.sh orqali chaqiriladi
	// Xizmat o'chganda/ishga tushganda darhol status yangilanadi
	r.Post("/api/internal/service-notify", HandleServiceNotify)

	// WebSocket
	r.Get("/ws", websocket.GlobalHub.HandleWebSocket)

	// Admin — redirect to home
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	})

	return r
}

// SpaHandler — React build'dan serve qiladi (web/frontend/dist/)
func SpaHandler(publicDir string) http.HandlerFunc {
	distDir := publicDir + "/frontend/dist"
	fs := http.FileServer(http.Dir(distDir))

	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Statik faylni tekshiramiz (JS, CSS, assets)
		path := distDir + r.URL.Path
		if _, err := os.Stat(path); err == nil && r.URL.Path != "/" {
			// Statik fayllar uchun cache header
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fs.ServeHTTP(w, r)
			return
		}

		// SPA fallback — index.html
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, distDir+"/index.html")
	}
}

