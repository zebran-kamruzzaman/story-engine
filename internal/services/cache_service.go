package services

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"story-engine/internal/models"

	"github.com/google/uuid"
)

// CacheService is the sole interface to the SQLite cache database.
// All read and write operations acquire the appropriate lock.
type CacheService struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewCacheService creates a CacheService wrapping the given database connection.
func NewCacheService(db *sql.DB) *CacheService {
	return &CacheService{db: db}
}

// GetAllScenes returns all scenes ordered by order_index ascending.
func (s *CacheService) GetAllScenes() ([]models.Scene, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT id, title, file_path, order_index, word_count, last_modified, cursor_position, scroll_top
		FROM scenes
		ORDER BY order_index ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("cache: get all scenes: %w", err)
	}
	defer rows.Close()

	var scenes []models.Scene
	for rows.Next() {
		var sc models.Scene
		if err := rows.Scan(
			&sc.ID, &sc.Title, &sc.FilePath, &sc.OrderIndex,
			&sc.WordCount, &sc.LastModified, &sc.CursorPosition, &sc.ScrollTop,
		); err != nil {
			return nil, fmt.Errorf("cache: scan scene row: %w", err)
		}
		scenes = append(scenes, sc)
	}
	return scenes, rows.Err()
}

// GetScene returns a single scene by ID.
func (s *CacheService) GetScene(id string) (models.Scene, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sc models.Scene
	err := s.db.QueryRow(`
		SELECT id, title, file_path, order_index, word_count, last_modified, cursor_position, scroll_top
		FROM scenes
		WHERE id = ?
	`, id).Scan(
		&sc.ID, &sc.Title, &sc.FilePath, &sc.OrderIndex,
		&sc.WordCount, &sc.LastModified, &sc.CursorPosition, &sc.ScrollTop,
	)
	if err == sql.ErrNoRows {
		return models.Scene{}, fmt.Errorf("cache: scene %q not found", id)
	}
	if err != nil {
		return models.Scene{}, fmt.Errorf("cache: get scene %q: %w", id, err)
	}
	return sc, nil
}

// CreateScene inserts a new scene record.
func (s *CacheService) CreateScene(scene models.Scene) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO scenes (id, title, file_path, order_index, word_count, last_modified, cursor_position, scroll_top)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, scene.ID, scene.Title, scene.FilePath, scene.OrderIndex,
		scene.WordCount, scene.LastModified, scene.CursorPosition, scene.ScrollTop)
	if err != nil {
		return fmt.Errorf("cache: create scene: %w", err)
	}
	return nil
}

// UpdateWordCount updates the word_count and last_modified timestamp for a scene.
func (s *CacheService) UpdateWordCount(id string, count int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE scenes SET word_count = ?, last_modified = ? WHERE id = ?
	`, count, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("cache: update word count for %q: %w", id, err)
	}
	return nil
}

// UpdateCursorState persists the cursor position and scroll offset for a scene.
func (s *CacheService) UpdateCursorState(id string, pos int, scrollTop float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		UPDATE scenes SET cursor_position = ?, scroll_top = ? WHERE id = ?
	`, pos, scrollTop, id)
	if err != nil {
		return fmt.Errorf("cache: update cursor state for %q: %w", id, err)
	}
	return nil
}

// RenameScene updates the title of a scene.
func (s *CacheService) RenameScene(id string, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`UPDATE scenes SET title = ? WHERE id = ?`, title, id)
	if err != nil {
		return fmt.Errorf("cache: rename scene %q: %w", id, err)
	}
	return nil
}

// DeleteScene removes a scene record and all associated entities.
// The ON DELETE CASCADE on the entities table handles child cleanup automatically.
func (s *CacheService) DeleteScene(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM scenes WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("cache: delete scene %q: %w", id, err)
	}
	return nil
}

// ReorderScene shifts a scene to a new order_index and adjusts all other scenes
// to maintain a contiguous ordering. This is done in a single transaction.
func (s *CacheService) ReorderScene(id string, newIndex int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("cache: begin reorder tx: %w", err)
	}
	defer tx.Rollback()

	// Shift all scenes at or above newIndex up by one to make room.
	if _, err := tx.Exec(`
		UPDATE scenes SET order_index = order_index + 1
		WHERE order_index >= ? AND id != ?
	`, newIndex, id); err != nil {
		return fmt.Errorf("cache: reorder shift: %w", err)
	}

	// Place the scene at its new position.
	if _, err := tx.Exec(`
		UPDATE scenes SET order_index = ? WHERE id = ?
	`, newIndex, id); err != nil {
		return fmt.Errorf("cache: reorder set: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache: reorder commit: %w", err)
	}
	return nil
}

// UpsertEntities replaces the complete entity list for a scene.
// Deletion of old entities is safe because of ON DELETE CASCADE.
func (s *CacheService) UpsertEntities(sceneID string, entities []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("cache: begin upsert entities tx: %w", err)
	}
	defer tx.Rollback()

	// Delete all existing entities for this scene.
	if _, err := tx.Exec(`DELETE FROM entities WHERE scene_id = ?`, sceneID); err != nil {
		return fmt.Errorf("cache: delete old entities: %w", err)
	}

	// Insert new entities.
	for _, name := range entities {
		if _, err := tx.Exec(`
			INSERT INTO entities (id, scene_id, name, frequency) VALUES (?, ?, ?, 1)
		`, uuid.New().String(), sceneID, name); err != nil {
			return fmt.Errorf("cache: insert entity %q: %w", name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cache: upsert entities commit: %w", err)
	}
	return nil
}

// GetEntities returns the list of detected entity names for a scene.
func (s *CacheService) GetEntities(sceneID string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`
		SELECT name FROM entities WHERE scene_id = ? ORDER BY frequency DESC
	`, sceneID)
	if err != nil {
		return nil, fmt.Errorf("cache: get entities for %q: %w", sceneID, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("cache: scan entity: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// UpsertMirror stores the interaction analysis for a scene.
// interactionsJSON must be a valid JSON array string.
func (s *CacheService) UpsertMirror(sceneID string, interactionsJSON string, sceneTone string, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`
		INSERT INTO scene_mirror (scene_id, interactions, scene_tone, source, updated_at)
		VALUES (?, ?, ?, ?, strftime('%s','now'))
		ON CONFLICT(scene_id) DO UPDATE SET
			interactions = excluded.interactions,
			scene_tone   = excluded.scene_tone,
			source       = excluded.source,
			updated_at   = excluded.updated_at
	`, sceneID, interactionsJSON, sceneTone, source)
	if err != nil {
		return fmt.Errorf("cache: upsert mirror for %q: %w", sceneID, err)
	}
	return nil
}

// GetMirror retrieves the stored interaction JSON, scene tone, and source for a scene.
// Returns safe empty values if no mirror data exists yet.
func (s *CacheService) GetMirror(sceneID string) (interactionsJSON string, sceneTone string, source string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	row := s.db.QueryRow(`
		SELECT interactions, scene_tone, source FROM scene_mirror WHERE scene_id = ?
	`, sceneID)
	err = row.Scan(&interactionsJSON, &sceneTone, &source)
	if err == sql.ErrNoRows {
		return "[]", "", "rule", nil
	}
	if err != nil {
		return "", "", "", fmt.Errorf("cache: get mirror for %q: %w", sceneID, err)
	}
	return interactionsJSON, sceneTone, source, nil
}

// GetMaxOrderIndex returns the current highest order_index across all scenes,
// used when appending a new scene to the end of the list.
func (s *CacheService) GetMaxOrderIndex() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var max int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(order_index), -1) FROM scenes`).Scan(&max)
	if err != nil {
		return 0, fmt.Errorf("cache: get max order index: %w", err)
	}
	return max, nil
}

// FilePathExists checks whether a scene with the given file_path already exists in the DB.
func (s *CacheService) FilePathExists(filePath string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM scenes WHERE file_path = ?`, filePath).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("cache: file path exists: %w", err)
	}
	return count > 0, nil
}
