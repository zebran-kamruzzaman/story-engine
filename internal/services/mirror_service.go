package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"story-engine/internal/models"
)

// MirrorService reads and writes LLM-generated data in the mirror/ directory.
// mirror/
//
//	characters.json     — project-wide character roster
//	chat.json           — project chat history
//	scenes/<id>.json    — per-scene summaries
type MirrorService struct {
	mirrorDir string
}

func NewMirrorService(mirrorDir string) *MirrorService {
	return &MirrorService{mirrorDir: mirrorDir}
}

// ── Character roster ──────────────────────────────────────────────────────────

func (s *MirrorService) GetCharacters() (map[string]models.CharacterProfile, error) {
	path := filepath.Join(s.mirrorDir, "characters.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]models.CharacterProfile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mirror: read characters: %w", err)
	}
	var roster map[string]models.CharacterProfile
	if err := json.Unmarshal(data, &roster); err != nil {
		return nil, fmt.Errorf("mirror: parse characters: %w", err)
	}
	return roster, nil
}

func (s *MirrorService) SaveCharacters(roster map[string]models.CharacterProfile) error {
	data, err := json.MarshalIndent(roster, "", "  ")
	if err != nil {
		return fmt.Errorf("mirror: marshal characters: %w", err)
	}
	return writeAtomic(filepath.Join(s.mirrorDir, "characters.json"), data)
}

// ── Scene summaries ───────────────────────────────────────────────────────────

func (s *MirrorService) GetSceneSummary(sceneID string) (models.SceneSummary, error) {
	path := filepath.Join(s.mirrorDir, "scenes", sceneID+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return models.SceneSummary{SceneID: sceneID}, nil
	}
	if err != nil {
		return models.SceneSummary{}, fmt.Errorf("mirror: read scene summary: %w", err)
	}
	var summary models.SceneSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return models.SceneSummary{}, fmt.Errorf("mirror: parse scene summary: %w", err)
	}
	return summary, nil
}

func (s *MirrorService) SaveSceneSummary(summary models.SceneSummary) error {
	dir := filepath.Join(s.mirrorDir, "scenes")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mirror: mkdir scenes: %w", err)
	}
	summary.UpdatedAt = time.Now().Unix()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return fmt.Errorf("mirror: marshal scene summary: %w", err)
	}
	return writeAtomic(filepath.Join(dir, summary.SceneID+".json"), data)
}

// ── Chat history ──────────────────────────────────────────────────────────────

func (s *MirrorService) GetChatHistory() ([]models.ChatMessage, error) {
	path := filepath.Join(s.mirrorDir, "chat.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []models.ChatMessage{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("mirror: read chat: %w", err)
	}
	var history []models.ChatMessage
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("mirror: parse chat: %w", err)
	}
	return history, nil
}

func (s *MirrorService) AppendChatMessages(msgs ...models.ChatMessage) error {
	history, err := s.GetChatHistory()
	if err != nil {
		history = []models.ChatMessage{}
	}
	history = append(history, msgs...)
	// Cap at 200 messages to prevent unbounded growth.
	if len(history) > 200 {
		history = history[len(history)-200:]
	}
	data, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("mirror: marshal chat: %w", err)
	}
	return writeAtomic(filepath.Join(s.mirrorDir, "chat.json"), data)
}

func (s *MirrorService) ClearChat() error {
	data, _ := json.Marshal([]models.ChatMessage{})
	return writeAtomic(filepath.Join(s.mirrorDir, "chat.json"), data)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// writeAtomic writes data to a file using a temp-file-then-rename pattern.
func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("mirror: write tmp %q: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("mirror: rename %q: %w", path, err)
	}
	return nil
}

// ParseSourcesFromResponse extracts scene titles mentioned after "Sources:" in an LLM answer.
// Returns the answer text (before Sources:) and the source list.
func ParseSourcesFromResponse(raw string, scenes []models.Scene) (answer string, sources []models.SceneSource) {
	idx := strings.Index(strings.ToLower(raw), "sources:")
	if idx < 0 {
		return strings.TrimSpace(raw), nil
	}
	answer = strings.TrimSpace(raw[:idx])
	rest := raw[idx+len("sources:"):]

	// Build a title→sceneID lookup.
	byTitle := make(map[string]models.Scene, len(scenes))
	for _, sc := range scenes {
		byTitle[strings.ToLower(strings.TrimSpace(sc.Title))] = sc
	}

	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "•")
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if sc, ok := byTitle[lower]; ok {
			sources = append(sources, models.SceneSource{SceneID: sc.ID, Title: sc.Title, Score: 1})
		}
	}
	return answer, sources
}
