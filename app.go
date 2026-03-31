package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"

	"story-engine/internal/db"
	"story-engine/internal/models"
	"story-engine/internal/services"
)

// App is the IPC coordinator. It owns all services and exposes methods to the
// frontend. Initialize() must be called from main.go before app.Run().
type App struct {
	mainWindow   *application.WebviewWindow
	file         *services.FileService
	cache        *services.CacheService
	analysis     *services.AnalysisService
	events       *services.EventService
	llm          *services.LLMService
	settings     models.AppSettings
	project      *models.Project
	ctx          context.Context
	cancel       context.CancelFunc
	settingsPath string
}

// Initialize sets up all services. It is called from main.go before app.Run()
// so that every service is ready before the WebView loads. This avoids any
// race between Wails service lifecycle timing and the frontend's first IPC call.
func (a *App) Initialize(win *application.WebviewWindow) error {
	a.mainWindow = win
	a.ctx, a.cancel = context.WithCancel(context.Background())

	// 1. Resolve project directory.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("initialize: home dir: %w", err)
	}
	projectDir := filepath.Join(home, "Documents", "StoryEngine", "default")
	scenesDir := filepath.Join(projectDir, "scenes")
	engineDir := filepath.Join(projectDir, ".engine")

	// 2. Ensure directories exist.
	for _, dir := range []string{scenesDir, engineDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("initialize: mkdir %q: %w", dir, err)
		}
	}

	// 3. Write project.json on first run.
	manifestPath := filepath.Join(projectDir, "project.json")
	if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
		proj := models.Project{Name: "default", CreatedAt: time.Now().Unix()}
		data, _ := json.MarshalIndent(proj, "", "  ")
		if writeErr := os.WriteFile(manifestPath, data, 0644); writeErr != nil {
			log.Printf("initialize: warn: could not write project.json: %v", writeErr)
		}
	}

	a.project = &models.Project{
		Name:      "default",
		CreatedAt: time.Now().Unix(),
		ScenesDir: scenesDir,
		EngineDir: engineDir,
	}

	// Load or create settings.json at ~/Documents/StoryEngine/settings.json.
	settingsPath := filepath.Join(home, "Documents", "StoryEngine", "settings.json")
	a.settings = models.DefaultSettings()
	if data, err := os.ReadFile(settingsPath); err == nil {
		if jsonErr := json.Unmarshal(data, &a.settings); jsonErr != nil {
			log.Printf("initialize: warn: could not parse settings.json: %v", jsonErr)
			a.settings = models.DefaultSettings()
		}
	}

	// 4. Initialize SQLite.
	dbPath := filepath.Join(engineDir, "project.db")
	database, err := db.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("initialize: db: %w", err)
	}

	// 5–8. Wire services.
	a.cache = services.NewCacheService(database)
	a.file = services.NewFileService(projectDir)
	a.events = services.NewEventService(win)
	a.analysis = services.NewAnalysisService(a.cache, a.events)
	a.llm = services.NewLLMService(a.settings)

	// 9. Start background analysis goroutine.
	a.analysis.Start(a.ctx)

	// 10. Sync disk ↔ DB on startup.
	a.syncOnStartup()

	log.Println("initialize: complete")
	return nil
}

// OnShutdown is called by Wails when the application closes.
func (a *App) OnShutdown() error {
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

// syncOnStartup reconciles .md files on disk with scene records in SQLite.
func (a *App) syncOnStartup() {
	files, err := a.file.ListSceneFiles(a.project.ScenesDir)
	if err != nil {
		log.Printf("sync: list scene files: %v", err)
		return
	}

	diskPaths := make(map[string]bool)
	for _, abs := range files {
		rel := filepath.Base(abs)
		diskPaths[rel] = true

		exists, err := a.cache.FilePathExists(rel)
		if err != nil {
			log.Printf("sync: check file path %q: %v", rel, err)
			continue
		}
		if !exists {
			title := strings.TrimSuffix(rel, ".md")
			maxIdx, _ := a.cache.GetMaxOrderIndex()
			scene := models.Scene{
				ID:           uuid.New().String(),
				Title:        title,
				FilePath:     rel,
				OrderIndex:   maxIdx + 1,
				LastModified: time.Now().Unix(),
			}
			if err := a.cache.CreateScene(scene); err != nil {
				log.Printf("sync: register scene %q: %v", rel, err)
			} else {
				log.Printf("sync: registered new scene %q", rel)
			}
		}
	}

	allScenes, err := a.cache.GetAllScenes()
	if err != nil {
		log.Printf("sync: get all scenes: %v", err)
		return
	}
	for _, sc := range allScenes {
		if !diskPaths[sc.FilePath] {
			if err := a.cache.DeleteScene(sc.ID); err != nil {
				log.Printf("sync: delete orphan scene %q: %v", sc.ID, err)
			} else {
				log.Printf("sync: deleted orphan record for %q", sc.FilePath)
			}
		}
	}
}

// ─────────────────────────────────────────────
// IPC Methods
// ─────────────────────────────────────────────

func (a *App) GetScenes() ([]models.Scene, error) {
	return a.cache.GetAllScenes()
}

func (a *App) GetSceneContent(id string) (string, error) {
	scene, err := a.cache.GetScene(id)
	if err != nil {
		return "", fmt.Errorf("GetSceneContent: %w", err)
	}
	return a.file.ReadScene(filepath.Join(a.project.ScenesDir, scene.FilePath))
}

func (a *App) SaveSceneContent(id string, content string) error {
	scene, err := a.cache.GetScene(id)
	if err != nil {
		return fmt.Errorf("SaveSceneContent: not found: %w", err)
	}
	if err := a.file.WriteScene(filepath.Join(a.project.ScenesDir, scene.FilePath), content); err != nil {
		return fmt.Errorf("SaveSceneContent: write: %w", err)
	}
	if err := a.cache.UpdateWordCount(id, countWords(content)); err != nil {
		log.Printf("warn: word count update failed for %s: %v", id, err)
	}
	a.analysis.Submit(services.AnalysisJob{SceneID: id, Content: content})
	return nil
}

func (a *App) CreateScene(title string) (models.Scene, error) {
	id := uuid.New().String()
	fileName := toSlug(title) + "_" + id[:8] + ".md"
	if err := a.file.WriteScene(filepath.Join(a.project.ScenesDir, fileName), ""); err != nil {
		return models.Scene{}, fmt.Errorf("CreateScene: write: %w", err)
	}
	maxIdx, err := a.cache.GetMaxOrderIndex()
	if err != nil {
		return models.Scene{}, fmt.Errorf("CreateScene: max order: %w", err)
	}
	scene := models.Scene{
		ID:           id,
		Title:        title,
		FilePath:     fileName,
		OrderIndex:   maxIdx + 1,
		LastModified: time.Now().Unix(),
	}
	if err := a.cache.CreateScene(scene); err != nil {
		return models.Scene{}, fmt.Errorf("CreateScene: cache: %w", err)
	}
	return scene, nil
}

func (a *App) RenameScene(id string, title string) error {
	return a.cache.RenameScene(id, title)
}

func (a *App) DeleteScene(id string) error {
	scene, err := a.cache.GetScene(id)
	if err != nil {
		return fmt.Errorf("DeleteScene: %w", err)
	}
	if err := a.cache.DeleteScene(id); err != nil {
		return fmt.Errorf("DeleteScene: db: %w", err)
	}
	fullPath := filepath.Join(a.project.ScenesDir, scene.FilePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		log.Printf("warn: DeleteScene: remove file %q: %v", fullPath, err)
	}
	return nil
}

func (a *App) ReorderScene(id string, newIndex int) error {
	return a.cache.ReorderScene(id, newIndex)
}

func (a *App) SaveCursorState(id string, cursorPos int, scrollTop float64) error {
	return a.cache.UpdateCursorState(id, cursorPos, scrollTop)
}

// GetInsights returns the last-known mirror state for a scene.
// Called when loading a scene so the panel shows previous results immediately.
func (a *App) GetInsights(sceneID string) (models.SceneInsights, error) {
	scene, err := a.cache.GetScene(sceneID)
	if err != nil {
		return models.SceneInsights{}, fmt.Errorf("GetInsights: %w", err)
	}
	entities, err := a.cache.GetEntities(sceneID)
	if err != nil {
		return models.SceneInsights{}, fmt.Errorf("GetInsights: entities: %w", err)
	}
	interactionsJSON, sceneTone, source, err := a.cache.GetMirror(sceneID)
	if err != nil {
		log.Printf("GetInsights: mirror: %v", err)
		interactionsJSON = "[]"
		sceneTone = ""
		source = "rule"
	}
	return models.SceneInsights{
		SceneID:          sceneID,
		Entities:         entities,
		InteractionsJSON: interactionsJSON,
		SceneTone:        sceneTone,
		Source:           source,
		WordCount:        scene.WordCount,
	}, nil
}

// AnalyzeScene sends the current scene content to the configured LLM and updates
// the Mirror panel with the result. This is called by the brain icon in InsightsPanel.
// It runs synchronously — the frontend awaits it and shows a loading state.
// On success, a "mirror:updated" event is emitted automatically so the panel updates.
func (a *App) AnalyzeScene(sceneID string) error {
	// Read current content from disk (not cache — file is always source of truth).
	scene, err := a.cache.GetScene(sceneID)
	if err != nil {
		return fmt.Errorf("AnalyzeScene: scene not found: %w", err)
	}
	content, err := a.file.ReadScene(filepath.Join(a.project.ScenesDir, scene.FilePath))
	if err != nil {
		return fmt.Errorf("AnalyzeScene: read file: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("AnalyzeScene: scene is empty")
	}

	// Call the LLM.
	result, err := a.llm.AnalyzeScene(content)
	if err != nil {
		return fmt.Errorf("AnalyzeScene: %w", err)
	}

	// Persist character list and mirror data.
	if err := a.cache.UpsertEntities(sceneID, result.Characters); err != nil {
		log.Printf("AnalyzeScene: upsert entities: %v", err)
	}

	// Encode interactions to JSON for storage.
	interactionsBytes, err := json.Marshal(result.Interactions)
	interactionsJSON := "[]"
	if err == nil {
		interactionsJSON = string(interactionsBytes)
	}
	if err := a.cache.UpsertMirror(sceneID, interactionsJSON, result.SceneTone, "llm"); err != nil {
		log.Printf("AnalyzeScene: upsert mirror: %v", err)
	}

	// Emit event to update the frontend immediately.
	a.events.EmitMirrorUpdated(sceneID, result.Characters, result.Interactions, result.SceneTone, "llm")

	return nil
}

// GetSettings returns the current LLM settings.
func (a *App) GetSettings() (models.AppSettings, error) {
	return a.settings, nil
}

// SaveSettings persists new LLM settings and hot-swaps the LLM service configuration.
// The writer does not need to restart the app after changing settings.
func (a *App) SaveSettings(settings models.AppSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveSettings: marshal: %w", err)
	}
	if err := os.WriteFile(a.settingsPath, data, 0644); err != nil {
		return fmt.Errorf("SaveSettings: write: %w", err)
	}
	a.settings = settings
	a.llm.UpdateConfig(settings)
	log.Printf("settings saved: endpoint=%s model=%s", settings.LLMEndpoint, settings.LLMModel)
	return nil
}

// ─────────────────────────────────
// Helpers
// ─────────────────────────────────

func countWords(s string) int {
	return len(strings.Fields(s))
}

func toSlug(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' {
			b.WriteRune('-')
		}
	}
	if s := b.String(); s != "" {
		return s
	}
	return "scene"
}
