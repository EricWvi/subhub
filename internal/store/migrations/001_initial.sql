CREATE TABLE IF NOT EXISTS providers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    url TEXT NOT NULL,
    refresh_interval_minutes INTEGER NOT NULL,
    abbrev TEXT NOT NULL DEFAULT '',
    used INTEGER NOT NULL DEFAULT 0,
    total INTEGER NOT NULL DEFAULT 0,
    expire INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS proxy_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    raw_yaml TEXT NOT NULL,
    update_mark INTEGER NOT NULL DEFAULT 0,
    UNIQUE(provider_id, name),
    FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS provider_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL UNIQUE,
    format TEXT NOT NULL,
    normalized_yaml TEXT NOT NULL,
    node_count INTEGER NOT NULL,
    fetched_at TEXT NOT NULL,
    FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS refresh_attempts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id INTEGER NOT NULL UNIQUE,
    status TEXT NOT NULL,
    message TEXT NOT NULL,
    attempted_at TEXT NOT NULL,
    FOREIGN KEY(provider_id) REFERENCES providers(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS proxy_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    script_text TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
