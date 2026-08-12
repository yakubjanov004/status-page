package api

import (
	"context"
	"net/http"
	"status-page/internal/config"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

// AuthMiddleware checks for a valid session cookie
func AuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session_token")
			if err != nil || cookie.Value == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Validate session token
			if cookie.Value != cfg.SessionSecret {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, cfg.AdminUsername)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SpaHandler serves the frontend SPA for any unmatched routes
func SpaHandler(publicDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// If it's an API route and we got here, it's a 404
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		
		// Admin paths
		if strings.HasPrefix(r.URL.Path, "/admin") {
			http.ServeFile(w, r, publicDir+"/admin/index.html")
			return
		}

		// Otherwise serve public status page
		http.ServeFile(w, r, publicDir+"/public/index.html")
	}
}
