package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status-page/internal/model"
)

// ---------- Integration Tests ----------
// These tests use a temporary SQLite file and exercise the full HTTP stack.

func TestIntegration_FullEventFlow(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	// 1. Service starts as unknown
	services := getServices(t, router)
	findService := func(name string) model.ServiceStatus {
		for _, s := range services {
			if s.Name == name {
				return s
			}
		}
		t.Fatalf("service %q not found", name)
		return model.ServiceStatus{}
	}

	datan := findService("Datan")
	if datan.LastStatus != "unknown" {
		t.Errorf("expected Datan status=unknown, got %s", datan.LastStatus)
	}

	// 2. Service goes down
	downTime := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	w := postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Datan",
		Action:  "down",
		Time:    downTime.Format(time.RFC3339),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("down webhook failed: %d %s", w.Code, w.Body.String())
	}

	// 3. Verify service status changed to down
	services = getServices(t, router)
	datan = findService("Datan")
	if datan.LastStatus != "down" {
		t.Errorf("expected Datan status=down, got %s", datan.LastStatus)
	}
	if datan.OpenIncidentsCount != 1 {
		t.Errorf("expected 1 open incident, got %d", datan.OpenIncidentsCount)
	}

	// 4. Service comes back up (30 minutes later)
	upTime := downTime.Add(30 * time.Minute)
	w = postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Datan",
		Action:  "up",
		Time:    upTime.Format(time.RFC3339),
	})
	if w.Code != http.StatusOK {
		t.Fatalf("up webhook failed: %d %s", w.Code, w.Body.String())
	}

	// 5. Verify service status changed to up
	services = getServices(t, router)
	datan = findService("Datan")
	if datan.LastStatus != "up" {
		t.Errorf("expected Datan status=up, got %s", datan.LastStatus)
	}
	if datan.OpenIncidentsCount != 0 {
		t.Errorf("expected 0 open incidents, got %d", datan.OpenIncidentsCount)
	}

	// 6. Verify incident details
	incResp := getIncidents(t, router, "Datan", 10, 0)
	if incResp.Total != 1 {
		t.Fatalf("expected 1 total incident, got %d", incResp.Total)
	}
	inc := incResp.Incidents[0]
	if inc.Status != "closed" {
		t.Errorf("expected closed, got %s", inc.Status)
	}
	if inc.DurationSeconds == nil || *inc.DurationSeconds != 1800 {
		var got int64
		if inc.DurationSeconds != nil {
			got = *inc.DurationSeconds
		}
		t.Errorf("expected duration=1800s (30min), got %d", got)
	}
}

func TestIntegration_IncidentsPagination(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	// Create 5 incidents for AlfaConnect
	for i := 0; i < 5; i++ {
		downTime := time.Date(2025, 1, 1+i, 10, 0, 0, 0, time.UTC)
		upTime := downTime.Add(10 * time.Minute)

		postWebhook(t, router, testToken, model.WebhookRequest{
			Service: "AlfaConnect",
			Action:  "down",
			Time:    downTime.Format(time.RFC3339),
		})
		postWebhook(t, router, testToken, model.WebhookRequest{
			Service: "AlfaConnect",
			Action:  "up",
			Time:    upTime.Format(time.RFC3339),
		})
	}

	// Page 1: limit=2, offset=0
	resp := getIncidents(t, router, "AlfaConnect", 2, 0)
	if resp.Total != 5 {
		t.Errorf("expected total=5, got %d", resp.Total)
	}
	if len(resp.Incidents) != 2 {
		t.Errorf("expected 2 incidents on page 1, got %d", len(resp.Incidents))
	}
	if resp.Limit != 2 || resp.Offset != 0 {
		t.Errorf("expected limit=2 offset=0, got limit=%d offset=%d", resp.Limit, resp.Offset)
	}

	// Page 2: limit=2, offset=2
	resp = getIncidents(t, router, "AlfaConnect", 2, 2)
	if len(resp.Incidents) != 2 {
		t.Errorf("expected 2 incidents on page 2, got %d", len(resp.Incidents))
	}

	// Page 3: limit=2, offset=4
	resp = getIncidents(t, router, "AlfaConnect", 2, 4)
	if len(resp.Incidents) != 1 {
		t.Errorf("expected 1 incident on page 3, got %d", len(resp.Incidents))
	}
}

func TestIntegration_UptimeCalculation(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	now := time.Now().UTC()

	// Create a 1-hour incident that ended 2 hours ago
	downTime := now.Add(-3 * time.Hour)
	upTime := now.Add(-2 * time.Hour)

	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Tokpoint",
		Action:  "down",
		Time:    downTime.Format(time.RFC3339),
	})
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Tokpoint",
		Action:  "up",
		Time:    upTime.Format(time.RFC3339),
	})

	// Check 24h uptime
	uptimeResp := getUptime(t, router, "Tokpoint", "24h")
	if uptimeResp.Window != "24h" {
		t.Errorf("expected window=24h, got %s", uptimeResp.Window)
	}

	// Downtime should be approximately 3600 seconds (1 hour)
	if uptimeResp.TotalDowntimeSeconds < 3550 || uptimeResp.TotalDowntimeSeconds > 3650 {
		t.Errorf("expected ~3600s downtime, got %d", uptimeResp.TotalDowntimeSeconds)
	}

	// Uptime should be approximately (24*3600 - 3600) / (24*3600) * 100 ≈ 95.83%
	expectedUptime := float64(24*3600-3600) / float64(24*3600) * 100.0
	if uptimeResp.UptimePercent < expectedUptime-1.0 || uptimeResp.UptimePercent > expectedUptime+1.0 {
		t.Errorf("expected uptime ~%.2f%%, got %.2f%%", expectedUptime, uptimeResp.UptimePercent)
	}
}

func TestIntegration_UptimeWithOpenIncident(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	now := time.Now().UTC()

	// Create an incident that is still open (started 1 hour ago)
	downTime := now.Add(-1 * time.Hour)

	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Odimrepo",
		Action:  "down",
		Time:    downTime.Format(time.RFC3339),
	})

	// Check 24h uptime — open incident should use now as end
	uptimeResp := getUptime(t, router, "Odimrepo", "24h")

	// Downtime should be approximately 3600 seconds (1 hour)
	if uptimeResp.TotalDowntimeSeconds < 3550 || uptimeResp.TotalDowntimeSeconds > 3650 {
		t.Errorf("expected ~3600s downtime for open incident, got %d", uptimeResp.TotalDowntimeSeconds)
	}
}

func TestIntegration_StatusPage(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html, got %s", ct)
	}
}

func TestIntegration_MultipleServicesIndependent(t *testing.T) {
	db := setupTestDB(t)
	router := setupTestRouter(db, testToken)

	now := time.Now().UTC()

	// Down AlfaConnect
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "down",
		Time:    now.Format(time.RFC3339),
	})

	// Down Mehmonxona
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "Mehmonxona",
		Action:  "down",
		Time:    now.Format(time.RFC3339),
	})

	// Up AlfaConnect (should NOT affect Mehmonxona)
	postWebhook(t, router, testToken, model.WebhookRequest{
		Service: "AlfaConnect",
		Action:  "up",
		Time:    now.Add(5 * time.Minute).Format(time.RFC3339),
	})

	// AlfaConnect should have 0 open incidents
	alfaInc := getIncidents(t, router, "AlfaConnect", 10, 0)
	if alfaInc.Total != 1 {
		t.Fatalf("expected 1 incident for AlfaConnect, got %d", alfaInc.Total)
	}
	if alfaInc.Incidents[0].Status != "closed" {
		t.Errorf("AlfaConnect incident should be closed")
	}

	// Mehmonxona should still have 1 open incident
	mehInc := getIncidents(t, router, "Mehmonxona", 10, 0)
	if mehInc.Total != 1 {
		t.Fatalf("expected 1 incident for Mehmonxona, got %d", mehInc.Total)
	}
	if mehInc.Incidents[0].Status != "open" {
		t.Errorf("Mehmonxona incident should still be open")
	}
}

// ---------- Helpers ----------

func getServices(t *testing.T, router http.Handler) []model.ServiceStatus {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/services", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /services failed: %d %s", w.Code, w.Body.String())
	}
	var services []model.ServiceStatus
	if err := json.Unmarshal(w.Body.Bytes(), &services); err != nil {
		t.Fatalf("failed to unmarshal services: %v", err)
	}
	return services
}

func getIncidents(t *testing.T, router http.Handler, name string, limit, offset int) model.PaginatedIncidentsResponse {
	t.Helper()
	url := fmt.Sprintf("/api/v1/services/%s/incidents?limit=%d&offset=%d", name, limit, offset)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s failed: %d %s", url, w.Code, w.Body.String())
	}
	var resp model.PaginatedIncidentsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal incidents: %v", err)
	}
	return resp
}

func getUptime(t *testing.T, router http.Handler, name, window string) model.UptimeResponse {
	t.Helper()
	url := fmt.Sprintf("/api/v1/services/%s/uptime?window=%s", name, window)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s failed: %d %s", url, w.Code, w.Body.String())
	}
	var resp model.UptimeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal uptime: %v", err)
	}
	return resp
}
