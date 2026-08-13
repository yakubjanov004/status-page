package webhookdb

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"status-page/internal/model"

	_ "modernc.org/sqlite"
)

// migrationSQL contains the webhook system schema.
// Inlined here because Go embed cannot cross package boundaries.
const migrationSQL = `
CREATE TABLE IF NOT EXISTS webhook_services (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS webhook_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER NOT NULL,
    action TEXT NOT NULL CHECK(action IN ('up', 'down')),
    time DATETIME NOT NULL,
    payload JSON,
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (service_id) REFERENCES webhook_services(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_service_time
    ON webhook_events(service_id, time);

CREATE INDEX IF NOT EXISTS idx_webhook_events_time
    ON webhook_events(time);

CREATE TABLE IF NOT EXISTS webhook_incidents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    service_id INTEGER NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NULL,
    duration_seconds INTEGER NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK(status IN ('open', 'closed')),
    created_at DATETIME NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (service_id) REFERENCES webhook_services(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_incidents_service_status
    ON webhook_incidents(service_id, status);

CREATE INDEX IF NOT EXISTS idx_webhook_incidents_service_time
    ON webhook_incidents(service_id, start_time);
`

// DB wraps a sql.DB connection for the webhook system.
type DB struct {
	conn *sql.DB
}

// KnownServices are the five services that must be pre-seeded on startup.
var KnownServices = []string{
	"AlfaConnect",
	"Mehmonxona",
	"Odimrepo",
	"Tokpoint",
	"Datan",
}

// Init opens the SQLite database, configures pragmas, runs migrations,
// and seeds the known services. Returns a DB handle.
func Init(dbPath string) (*DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	conn, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// SQLite writes are serialized; keep one connection to avoid SQLITE_BUSY
	conn.SetMaxOpenConns(1)
	conn.SetConnMaxLifetime(0)

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	d := &DB{conn: conn}

	if err := d.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	if err := d.seedServices(); err != nil {
		return nil, fmt.Errorf("failed to seed services: %w", err)
	}

	log.Printf("[webhookdb] initialized database at %s", dbPath)
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.conn.Close()
}

// Ping checks if the database is reachable.
func (d *DB) Ping() error {
	return d.conn.Ping()
}

// Conn returns the underlying *sql.DB (used by tests).
func (d *DB) Conn() *sql.DB {
	return d.conn
}

func (d *DB) runMigrations() error {
	queries := strings.Split(migrationSQL, ";")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		if _, err := d.conn.Exec(query); err != nil {
			// Idempotent: ignore "already exists" errors
			errStr := err.Error()
			if strings.Contains(errStr, "already exists") ||
				strings.Contains(errStr, "duplicate column name") {
				continue
			}
			return fmt.Errorf("migration query failed: %w\nQuery: %s", err, query)
		}
	}

	log.Println("[webhookdb] migrations applied successfully")
	return nil
}

func (d *DB) seedServices() error {
	stmt, err := d.conn.Prepare(`INSERT OR IGNORE INTO webhook_services (name, created_at) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, name := range KnownServices {
		if _, err := stmt.Exec(name, now); err != nil {
			return fmt.Errorf("failed to seed service %s: %w", name, err)
		}
	}

	log.Printf("[webhookdb] seeded %d known services", len(KnownServices))
	return nil
}

// ---------- Service queries ----------

// GetServiceByName retrieves a service by name.
func (d *DB) GetServiceByName(name string) (*model.Service, error) {
	var s model.Service
	var createdStr string
	err := d.conn.QueryRow(
		`SELECT id, name, created_at FROM webhook_services WHERE name = ?`, name,
	).Scan(&s.ID, &s.Name, &createdStr)
	if err != nil {
		return nil, err
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdStr)
	return &s, nil
}

// ListServices returns all services with their last status and open incident count.
func (d *DB) ListServices() ([]model.ServiceStatus, error) {
	rows, err := d.conn.Query(`
		SELECT
			s.id,
			s.name,
			COALESCE(
				(SELECT e.action FROM webhook_events e
				 WHERE e.service_id = s.id ORDER BY e.time DESC LIMIT 1),
				'unknown'
			) AS last_status,
			(SELECT e.time FROM webhook_events e
			 WHERE e.service_id = s.id ORDER BY e.time DESC LIMIT 1) AS last_seen,
			(SELECT COUNT(*) FROM webhook_incidents i
			 WHERE i.service_id = s.id AND i.status = 'open') AS open_incidents_count
		FROM webhook_services s
		ORDER BY s.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.ServiceStatus
	for rows.Next() {
		var ss model.ServiceStatus
		var lastSeen sql.NullString
		if err := rows.Scan(&ss.ID, &ss.Name, &ss.LastStatus, &lastSeen, &ss.OpenIncidentsCount); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			ss.LastSeen = &lastSeen.String
		}
		results = append(results, ss)
	}
	return results, rows.Err()
}

// ---------- Event recording ----------

// RecordEvent inserts an event and creates/closes incidents atomically.
// Writes are serialized via MaxOpenConns(1), so no SQLITE_BUSY races.
func (d *DB) RecordEvent(serviceName, action string, eventTime time.Time, payload string) error {
	tx, err := d.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	// 1. Resolve service ID
	var serviceID int64
	err = tx.QueryRow(`SELECT id FROM webhook_services WHERE name = ?`, serviceName).Scan(&serviceID)
	if err != nil {
		return fmt.Errorf("unknown service %q: %w", serviceName, err)
	}

	// 2. Insert event
	timeStr := eventTime.UTC().Format(time.RFC3339)
	nowStr := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.Exec(
		`INSERT INTO webhook_events (service_id, action, time, payload, created_at) VALUES (?, ?, ?, ?, ?)`,
		serviceID, action, timeStr, payload, nowStr,
	)
	if err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}

	// 3. Handle incident logic
	switch action {
	case "down":
		// Check for existing open incident
		var exists int
		err = tx.QueryRow(
			`SELECT COUNT(*) FROM webhook_incidents WHERE service_id = ? AND status = 'open'`,
			serviceID,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check open incidents: %w", err)
		}
		if exists == 0 {
			// Create new incident
			_, err = tx.Exec(
				`INSERT INTO webhook_incidents (service_id, start_time, status, created_at) VALUES (?, ?, 'open', ?)`,
				serviceID, timeStr, nowStr,
			)
			if err != nil {
				return fmt.Errorf("failed to create incident: %w", err)
			}
			log.Printf("[webhookdb] incident opened for service %q at %s", serviceName, timeStr)
		} else {
			log.Printf("[webhookdb] ignoring duplicate down for service %q (incident already open)", serviceName)
		}

	case "up":
		// Find open incident to close
		var incidentID int64
		var startTimeStr string
		err = tx.QueryRow(
			`SELECT id, start_time FROM webhook_incidents WHERE service_id = ? AND status = 'open' LIMIT 1`,
			serviceID,
		).Scan(&incidentID, &startTimeStr)
		if err == sql.ErrNoRows {
			log.Printf("[webhookdb] ignoring duplicate up for service %q (no open incident)", serviceName)
		} else if err != nil {
			return fmt.Errorf("failed to find open incident: %w", err)
		} else {
			// Close the incident
			startTime, _ := time.Parse(time.RFC3339, startTimeStr)
			durationSec := int64(eventTime.UTC().Sub(startTime).Seconds())
			if durationSec < 0 {
				durationSec = 0
			}
			_, err = tx.Exec(
				`UPDATE webhook_incidents SET end_time = ?, duration_seconds = ?, status = 'closed' WHERE id = ?`,
				timeStr, durationSec, incidentID,
			)
			if err != nil {
				return fmt.Errorf("failed to close incident: %w", err)
			}
			log.Printf("[webhookdb] incident %d closed for service %q, duration=%ds", incidentID, serviceName, durationSec)
		}
	}

	return tx.Commit()
}

// ---------- Incident queries ----------

// ListIncidents returns incidents for a service with pagination.
func (d *DB) ListIncidents(serviceName string, limit, offset int) (*model.PaginatedIncidentsResponse, error) {
	svc, err := d.GetServiceByName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service %q not found: %w", serviceName, err)
	}

	// Count total
	var total int
	err = d.conn.QueryRow(
		`SELECT COUNT(*) FROM webhook_incidents WHERE service_id = ?`, svc.ID,
	).Scan(&total)
	if err != nil {
		return nil, err
	}

	rows, err := d.conn.Query(`
		SELECT id, start_time, end_time, duration_seconds, status
		FROM webhook_incidents
		WHERE service_id = ?
		ORDER BY start_time DESC
		LIMIT ? OFFSET ?
	`, svc.ID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []model.IncidentResponse
	for rows.Next() {
		var inc model.IncidentResponse
		var endTime sql.NullString
		var duration sql.NullInt64
		var startTimeStr string

		if err := rows.Scan(&inc.ID, &startTimeStr, &endTime, &duration, &inc.Status); err != nil {
			return nil, err
		}
		inc.ServiceName = serviceName
		inc.StartTime = startTimeStr
		if endTime.Valid {
			inc.EndTime = &endTime.String
		}
		if duration.Valid {
			inc.DurationSeconds = &duration.Int64
		}
		incidents = append(incidents, inc)
	}
	if incidents == nil {
		incidents = []model.IncidentResponse{}
	}

	return &model.PaginatedIncidentsResponse{
		Incidents: incidents,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}, rows.Err()
}

// ---------- Uptime calculation ----------

// ComputeUptime calculates uptime percentage and total downtime for a service
// over the given window. Incidents are clipped to window boundaries.
// Open incidents use now() as their effective end time.
func (d *DB) ComputeUptime(serviceName string, window time.Duration) (*model.UptimeResponse, error) {
	svc, err := d.GetServiceByName(serviceName)
	if err != nil {
		return nil, fmt.Errorf("service %q not found: %w", serviceName, err)
	}

	now := time.Now().UTC()
	windowStart := now.Add(-window)
	windowSeconds := int64(window.Seconds())

	// Single query: all incidents overlapping [windowStart, now]
	// Condition: started before now AND (ended after windowStart OR still open)
	rows, err := d.conn.Query(`
		SELECT start_time, end_time
		FROM webhook_incidents
		WHERE service_id = ?
		  AND start_time < ?
		  AND (end_time > ? OR end_time IS NULL)
		ORDER BY start_time ASC
	`, svc.ID, now.Format(time.RFC3339), windowStart.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var totalDowntime int64

	for rows.Next() {
		var startTimeStr string
		var endTimeStr sql.NullString

		if err := rows.Scan(&startTimeStr, &endTimeStr); err != nil {
			return nil, err
		}

		incStart, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			continue
		}

		var incEnd time.Time
		if endTimeStr.Valid {
			incEnd, err = time.Parse(time.RFC3339, endTimeStr.String)
			if err != nil {
				continue
			}
		} else {
			incEnd = now
		}

		// Clip to window boundaries
		if incStart.Before(windowStart) {
			incStart = windowStart
		}
		if incEnd.After(now) {
			incEnd = now
		}

		dur := int64(incEnd.Sub(incStart).Seconds())
		if dur > 0 {
			totalDowntime += dur
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	uptimePct := 100.0
	if windowSeconds > 0 && totalDowntime > 0 {
		uptimePct = float64(windowSeconds-totalDowntime) / float64(windowSeconds) * 100.0
		if uptimePct < 0 {
			uptimePct = 0
		}
	}

	return &model.UptimeResponse{
		ServiceName:          svc.Name,
		Window:               formatWindow(window),
		UptimePercent:        uptimePct,
		TotalDowntimeSeconds: totalDowntime,
	}, nil
}

func formatWindow(d time.Duration) string {
	switch {
	case d == 24*time.Hour:
		return "24h"
	case d == 7*24*time.Hour:
		return "7d"
	case d == 30*24*time.Hour:
		return "30d"
	default:
		return d.String()
	}
}
