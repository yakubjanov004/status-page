package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"status-page/internal/model"
	"status-page/internal/webhookdb"

	"github.com/go-chi/chi/v5"
)

// contextKey is a private type for context keys in this package.
type contextKey string

const ctxRequestID contextKey = "request_id"

// Handler holds the webhook HTTP handlers and their dependencies.
type Handler struct {
	db        *webhookdb.DB
	hookToken string
}

// New creates a new Handler.
func New(db *webhookdb.DB, hookToken string) *Handler {
	return &Handler{db: db, hookToken: hookToken}
}

// maxBodySize limits request body to 64KB to prevent abuse.
const maxBodySize = 64 * 1024

// maxServiceNameLen limits service name length in webhook requests.
const maxServiceNameLen = 64

// ---------- Middleware ----------

// requestIDMiddleware reads X-Request-Id from the incoming request (if present)
// or generates a short random ID, then stores it in the context and echoes it
// in the X-Request-Id response header for end-to-end traceability.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-Id")
		if reqID == "" {
			reqID = fmt.Sprintf("wh-%08x", rand.Uint32())
		}
		w.Header().Set("X-Request-Id", reqID)
		ctx := context.WithValue(r.Context(), ctxRequestID, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// getRequestID retrieves the request ID from the context.
func getRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(ctxRequestID).(string); ok {
		return id
	}
	return ""
}

// requireToken validates the X-Hook-Token header.
func (h *Handler) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Hook-Token")
		if token == "" || token != h.hookToken {
			writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
				Error:     "unauthorized: invalid or missing X-Hook-Token",
				RequestID: getRequestID(r),
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- POST /api/v1/webhook ----------

// HandleWebhook processes incoming service up/down events.
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	reqID := getRequestID(r)

	// Limit body size
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	var req model.WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[webhook] %s: invalid JSON body: %v", reqID, err)
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:     "invalid JSON body",
			RequestID: reqID,
		})
		return
	}

	// Validate service name
	if req.Service == "" || len(req.Service) > maxServiceNameLen {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:     "service name is required and must be <= 64 characters",
			RequestID: reqID,
		})
		return
	}

	// Validate action
	if req.Action != "up" && req.Action != "down" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:     `action must be "up" or "down"`,
			RequestID: reqID,
		})
		return
	}

	// Parse time strictly as ISO8601 UTC
	eventTime, err := parseISO8601UTC(req.Time)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error:     fmt.Sprintf("invalid time format (expected ISO8601 UTC): %v", err),
			RequestID: reqID,
		})
		return
	}

	// Serialize meta to JSON string
	var payload string
	if req.Meta != nil {
		metaBytes, _ := json.Marshal(req.Meta)
		payload = string(metaBytes)
	}

	// Record event (creates/closes incidents atomically)
	if err := h.db.RecordEvent(req.Service, req.Action, eventTime, payload); err != nil {
		log.Printf("[webhook] %s: failed to record event for %q: %v", reqID, req.Service, err)
		// Unknown service name is a client error (400), not a server error (500)
		if errors.Is(err, webhookdb.ErrUnknownService) {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error:     fmt.Sprintf("unknown service %q: not in the list of known services", req.Service),
				RequestID: reqID,
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error:     "failed to record event",
			RequestID: reqID,
		})
		return
	}

	log.Printf("[webhook] %s: recorded %s event for service %q at %s", reqID, req.Action, req.Service, req.Time)
	writeJSON(w, http.StatusOK, model.SuccessResponse{
		Status:    "ok",
		Message:   fmt.Sprintf("event %s recorded for %s", req.Action, req.Service),
		RequestID: reqID,
	})
}

// ---------- GET /api/v1/services ----------

// HandleListServices returns all services with their status.
func (h *Handler) HandleListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		log.Printf("[handler] failed to list services: %v", err)
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: "failed to list services",
		})
		return
	}
	if services == nil {
		services = []model.ServiceStatus{}
	}
	writeJSON(w, http.StatusOK, services)
}

// ---------- GET /api/v1/services/{name}/incidents ----------

// HandleListIncidents returns paginated incidents for a service.
func (h *Handler) HandleListIncidents(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "service name required"})
		return
	}

	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	result, err := h.db.ListIncidents(name, limit, offset)
	if err != nil {
		log.Printf("[handler] failed to list incidents for %q: %v", name, err)
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{
			Error: fmt.Sprintf("service %q not found or query error", name),
		})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ---------- GET /api/v1/services/{name}/uptime ----------

// HandleUptime returns uptime stats for a service within a time window.
func (h *Handler) HandleUptime(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{Error: "service name required"})
		return
	}

	windowStr := r.URL.Query().Get("window")
	window, err := parseWindow(windowStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: fmt.Sprintf("invalid window parameter: %v (use 24h, 7d, or 30d)", err),
		})
		return
	}

	result, err := h.db.ComputeUptime(name, window)
	if err != nil {
		log.Printf("[handler] failed to compute uptime for %q: %v", name, err)
		writeJSON(w, http.StatusNotFound, model.ErrorResponse{
			Error: fmt.Sprintf("service %q not found or query error", name),
		})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ---------- GET /healthz ----------

// HandleHealthz returns 200 OK if the DB is reachable.
func (h *Handler) HandleHealthz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, model.ErrorResponse{
			Error: "database unreachable",
		})
		return
	}
	writeJSON(w, http.StatusOK, model.SuccessResponse{Status: "ok"})
}

// ---------- GET /status ----------

// HandleStatusPage renders a simple HTML status page.
func (h *Handler) HandleStatusPage(w http.ResponseWriter, r *http.Request) {
	services, err := h.db.ListServices()
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	fmt.Fprint(w, `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Service Status</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #0f172a; color: #e2e8f0; }
    h1 { color: #f8fafc; border-bottom: 2px solid #334155; padding-bottom: 16px; }
    .service { display: flex; align-items: center; justify-content: space-between; padding: 16px; margin: 8px 0; background: #1e293b; border-radius: 8px; border: 1px solid #334155; }
    .service-name { font-weight: 600; font-size: 1.1em; }
    .status { padding: 4px 12px; border-radius: 12px; font-size: 0.85em; font-weight: 500; }
    .status-up { background: #065f46; color: #6ee7b7; }
    .status-down { background: #7f1d1d; color: #fca5a5; }
    .status-unknown { background: #374151; color: #9ca3af; }
    .meta { font-size: 0.85em; color: #94a3b8; margin-top: 4px; }
    .footer { margin-top: 32px; text-align: center; color: #64748b; font-size: 0.85em; }
  </style>
</head>
<body>
  <h1>📊 Service Status</h1>`)

	for _, s := range services {
		statusClass := "status-" + s.LastStatus
		lastSeen := "never"
		if s.LastSeen != nil {
			lastSeen = *s.LastSeen
		}
		fmt.Fprintf(w, `
  <div class="service">
    <div>
      <div class="service-name">%s</div>
      <div class="meta">Last seen: %s | Open incidents: %d</div>
    </div>
    <span class="status %s">%s</span>
  </div>`, s.Name, lastSeen, s.OpenIncidentsCount, statusClass, s.LastStatus)
	}

	fmt.Fprintf(w, `
  <div class="footer">Updated: %s UTC</div>
</body>
</html>`, time.Now().UTC().Format("2006-01-02 15:04:05"))
}

// ---------- Helpers ----------

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

// parseISO8601UTC parses a time string as ISO8601 and ensures it is UTC.
func parseISO8601UTC(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("time is required")
	}

	// Try common ISO8601 formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05+00:00",
	}

	for _, f := range formats {
		t, err := time.Parse(f, s)
		if err == nil {
			return t.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as ISO8601 UTC", s)
}

// parseWindow converts window string (24h, 7d, 30d) to time.Duration.
func parseWindow(s string) (time.Duration, error) {
	switch s {
	case "24h":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	case "":
		return 24 * time.Hour, nil // default to 24h
	default:
		return 0, fmt.Errorf("unsupported window: %q", s)
	}
}
