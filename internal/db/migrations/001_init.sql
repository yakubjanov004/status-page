-- Monitorlar (tekshiriladigan servislar)
CREATE TABLE IF NOT EXISTS monitors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,                 -- 'http', 'tcp', 'ping'
    url TEXT NOT NULL,                  -- tekshiriladigan manzil
    interval_seconds INTEGER DEFAULT 60,-- necha soniyada tekshirilsin
    timeout_seconds INTEGER DEFAULT 10,
    expected_status_code INTEGER DEFAULT 200,
    is_active BOOLEAN DEFAULT 1,
    show_on_public_page BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Har bir tekshiruv natijasi (tarix)
CREATE TABLE IF NOT EXISTS heartbeats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    monitor_id INTEGER NOT NULL,
    is_up BOOLEAN NOT NULL,
    response_time_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (monitor_id) REFERENCES monitors(id)
);
CREATE INDEX IF NOT EXISTS idx_heartbeats_monitor_time ON heartbeats(monitor_id, checked_at);

-- Admin foydalanuvchilar
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Bildirishnomalar sozlamalari (Telegram, Discord va h.k.)
CREATE TABLE IF NOT EXISTS notifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,                 -- 'telegram', 'discord', 'email'
    config TEXT NOT NULL,               -- JSON: bot token, chat id va h.k.
    is_active BOOLEAN DEFAULT 1
);
