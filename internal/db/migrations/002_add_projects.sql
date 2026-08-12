-- Yangi jadval: projects
CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    description TEXT,
    show_on_public_page BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- monitors jadvaliga ustunlar qo'shish (SQLite'da ALTER TABLE orqali default bilan)
ALTER TABLE monitors ADD COLUMN project_id INTEGER REFERENCES projects(id);
ALTER TABLE monitors ADD COLUMN component_type TEXT DEFAULT 'backend';
