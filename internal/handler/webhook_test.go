package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"status-page/internal/model"
	"status-page/internal/webhookdb"

	"github.com/go-chi/chi/v5"
)


// setupTestDB creates a temporary SQLite database for testing.
func setupTestDB(t *testing.T) *webhookdb.DB {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := webhookdb.Init(dbPath)
	if err != nil {
		t.Fatalf("failed to init test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// setupTestRouter creates a chi router with all webhook routes for testing.
// hmacSecret="" disables HMAC verification (token-only auth).
func setupTestRouter(db *webhookdb.DB, token string) *chi.Mux {
	r := chi.NewRouter()
	SetupRoutes(r, db, token, "" /* hmacSecret: disabled in tests */)
	return r
}


const testToken = "test-token-secret"

func postWebhook(t *testing.T, router http.Handler, token string, req model.WebhookRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/api/v1/webhook", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	if token != "" {
		httpReq.Header.Set("X-Hook-Token", token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httpReq)
	return w
}

// ---------- Unit Tests ----------

func TestWebhook_Unauthorized(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	// No token
	w := postWebhook(t, router, "", model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "down",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}

	// Wrong token
	w = postWebhook(t, router, "wrong-token", model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "down",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWebhook_InvalidAction(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "restart", // Invalid
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhook_InvalidTime(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "down",
		Time:    "not-a-date",
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWebhook_DownCreatesIncident(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	now := time.Now().UTC()

	// Send DOWN
	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "down",
		Time:    now.Format(time.RFC3339),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify incident was created
	incidents, err := db.ListIncidents("AlfaConnect", 10, 0)
	if err != nil {
		t.Fatalf("failed to list incidents: %v", err)
	}
	if len(incidents.Incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents.Incidents))
	}
	if incidents.Incidents[0].Status != "open" {
		t.Errorf("expected open incident, got %s", incidents.Incidents[0].Status)
	}
	if incidents.Incidents[0].EndTime != nil {
		t.Errorf("expected nil end_time, got %v", incidents.Incidents[0].EndTime)
	}
}

func TestWebhook_UpClosesIncident(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	downTime := time.Now().UTC().Add(-5 * time.Minute)
	upTime := time.Now().UTC()

	// Send DOWN
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Mehmonxona",
		Action:  "down",
		Time:    downTime.Format(time.RFC3339),
	})

	// Send UP
	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Mehmonxona",
		Action:  "up",
		Time:    upTime.Format(time.RFC3339),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify incident was closed
	incidents, err := db.ListIncidents("Mehmonxona", 10, 0)
	if err != nil {
		t.Fatalf("failed to list incidents: %v", err)
	}
	if len(incidents.Incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents.Incidents))
	}
	inc := incidents.Incidents[0]
	if inc.Status != "closed" {
		t.Errorf("expected closed incident, got %s", inc.Status)
	}
	if inc.EndTime == nil {
		t.Fatal("expected end_time to be set")
	}
	if inc.DurationSeconds == nil {
		t.Fatal("expected duration_seconds to be set")
	}
	// Duration should be approximately 300 seconds (5 minutes)
	if *inc.DurationSeconds < 290 || *inc.DurationSeconds > 310 {
		t.Errorf("expected duration ~300s, got %d", *inc.DurationSeconds)
	}
}

func TestWebhook_DuplicateDownIgnored(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	now := time.Now().UTC()

	// Send DOWN twice
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Odimrepo",
		Action:  "down",
		Time:    now.Format(time.RFC3339),
	})
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Odimrepo",
		Action:  "down",
		Time:    now.Add(1 * time.Minute).Format(time.RFC3339),
	})

	// Should still have only 1 incident
	incidents, err := db.ListIncidents("Odimrepo", 10, 0)
	if err != nil {
		t.Fatalf("failed to list incidents: %v", err)
	}
	if len(incidents.Incidents) != 1 {
		t.Errorf("expected 1 incident (duplicate down ignored), got %d", len(incidents.Incidents))
	}
}

func TestWebhook_DuplicateUpIgnored(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	now := time.Now().UTC()

	// Send UP without any prior DOWN — should be ignored
	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Tokpoint",
		Action:  "up",
		Time:    now.Format(time.RFC3339),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Should have 0 incidents
	incidents, err := db.ListIncidents("Tokpoint", 10, 0)
	if err != nil {
		t.Fatalf("failed to list incidents: %v", err)
	}
	if len(incidents.Incidents) != 0 {
		t.Errorf("expected 0 incidents, got %d", len(incidents.Incidents))
	}
}

func TestWebhook_UnknownService(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "NonExistentService",
		Action:  "down",
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
	// Unknown service is a client error: webhook sender must fix the name
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown service, got %d", w.Code)
	}
	// Verify the error message is helpful
	body := w.Body.String()
	if !strings.Contains(body, "NonExistentService") {
		t.Errorf("expected error body to contain service name, got: %s", body)
	}
}

func TestHealthz(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestListServices(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var services []model.ServiceStatus
	if err := json.Unmarshal(w.Body.Bytes(), &services); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Should have all 5 seeded services
	if len(services) != 5 {
		t.Errorf("expected 5 services, got %d", len(services))
	}

	// All should start as "unknown"
	for _, s := range services {
		if s.LastStatus != "unknown" {
			t.Errorf("expected unknown status for %s, got %s", s.Name, s.LastStatus)
		}
	}
}

func TestMain(m *testing.M) {
	// Suppress log output during tests
	os.Exit(m.Run())
}
