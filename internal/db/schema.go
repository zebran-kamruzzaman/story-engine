package db

const pragmas = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;
`

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

CREATE TABLE IF NOT EXISTS scene_mirror (
    scene_id        TEXT    PRIMARY KEY REFERENCES scenes(id) ON DELETE CASCADE,
    interactions    TEXT    NOT NULL DEFAULT '[]',
    scene_tone      TEXT    NOT NULL DEFAULT '',
    source          TEXT    NOT NULL DEFAULT 'rule',
    updated_at      INTEGER NOT NULL DEFAULT 0
);
`
