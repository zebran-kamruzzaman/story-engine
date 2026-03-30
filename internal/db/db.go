package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver; registers "sqlite" with database/sql
)

// Initialize opens the SQLite database at the given path, applies session pragmas,
// and creates all tables if they do not already exist.
// The returned *sql.DB is ready for use and must be closed when the application exits.
func Initialize(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("db: open %q: %w", path, err)
	}

	// Apply pragmas. These are session-scoped and must be set on every connection.
	if _, err := db.Exec(pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("db: apply pragmas: %w", err)
	}

	// Create tables. Idempotent — safe on an existing database.
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("db: apply schema: %w", err)
	}

	return db, nil
}
