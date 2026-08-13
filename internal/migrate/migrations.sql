-- Webhook system tables (prefixed to avoid collision with existing schema)
-- This migration is idempotent: safe to run multiple times.

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
