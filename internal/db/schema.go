package db

// Pragmas are applied immediately after the connection is opened.
// They configure SQLite's behavior for this session.
const pragmas = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
`

// schema is the full DDL for the application's tables and indexes.
// Every CREATE statement uses IF NOT EXISTS, making initialization idempotent —
// safe to call on an existing database without wiping data.
const schema = `
CREATE TABLE IF NOT EXISTS scenes (
    id              TEXT    PRIMARY KEY,
    title           TEXT    NOT NULL,
    file_path       TEXT    NOT NULL UNIQUE,
    order_index     INTEGER NOT NULL DEFAULT 0,
    word_count      INTEGER NOT NULL DEFAULT 0,
    last_modified   INTEGER NOT NULL DEFAULT 0,
    cursor_position INTEGER NOT NULL DEFAULT 0,
    scroll_top      REAL    NOT NULL DEFAULT 0.0
);

CREATE TABLE IF NOT EXISTS entities (
    id        TEXT PRIMARY KEY,
    scene_id  TEXT NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    name      TEXT NOT NULL,
    frequency INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_entities_scene ON entities(scene_id);
`
