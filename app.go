package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"

	"story-engine/internal/db"
	"story-engine/internal/models"
	"story-engine/internal/services"
)

// App is the IPC coordinator. Initialize() must be called from main.go before app.Run().
type App struct {
	mainWindow   *application.WebviewWindow
	file         *services.FileService
	cache        *services.CacheService
	analysis     *services.AnalysisService
	events       *services.EventService
	llm          *services.LLMService
	mirror       *services.MirrorService
	project      *models.Project
	settings     models.AppSettings
	settingsPath string
	database     interface{ Close() error } // *sql.DB, closed on project switch

	ctx    context.Context
	cancel context.CancelFunc

	// Word-count threshold for automatic scene analysis.
	// Maps sceneID → word count at the time of the last LLM scene analysis.
	// Protected by waMu.
	wordCountAtLastAnalysis map[string]int
	waMu                    sync.Mutex
}

// ─── Initialization ─────────────────────────────────────────────────────────

// Initialize is called from main.go before app.Run().
func (a *App) Initialize(win *application.WebviewWindow) error {
	a.mainWindow = win
	a.wordCountAtLastAnalysis = make(map[string]int)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("initialize: home dir: %w", err)
	}
	storyEngineRoot := filepath.Join(home, "Documents", "StoryEngine")
	if err := os.MkdirAll(storyEngineRoot, 0755); err != nil {
		return fmt.Errorf("initialize: mkdir root: %w", err)
	}

	// Load or create global app settings.
	a.settingsPath = filepath.Join(storyEngineRoot, "app.json")
	a.settings = models.DefaultSettings()
	if data, err := os.ReadFile(a.settingsPath); err == nil {
		if jsonErr := json.Unmarshal(data, &a.settings); jsonErr != nil {
			log.Printf("initialize: warn: parse app.json: %v", jsonErr)
			a.settings = models.DefaultSettings()
		}
	}

	a.llm = services.NewLLMService(a.settings)

	// Determine which project to open.
	projectPath := a.settings.LastProjectPath
	if projectPath == "" || !dirExists(projectPath) {
		// First run or last project missing — open/create "default".
		projectPath = filepath.Join(storyEngineRoot, "default")
	}

	if err := a.loadProject(projectPath); err != nil {
		return fmt.Errorf("initialize: load project: %w", err)
	}
	return nil
}

// loadProject tears down any running project state and initializes a new one.
// It is safe to call multiple times (project switching).
func (a *App) loadProject(projectPath string) error {
	// Stop previous analysis goroutine.
	if a.cancel != nil {
		a.cancel()
	}

	// Close previous database.
	if a.database != nil {
		if err := a.database.Close(); err != nil {
			log.Printf("loadProject: close db: %v", err)
		}
		a.database = nil
	}

	// Ensure project directory structure.
	scenesDir := filepath.Join(projectPath, "scenes")
	mirrorDir := filepath.Join(projectPath, "mirror")
	mirrorScenesDir := filepath.Join(mirrorDir, "scenes")
	engineDir := filepath.Join(projectPath, ".engine")

	for _, dir := range []string{scenesDir, mirrorDir, mirrorScenesDir, engineDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("loadProject: mkdir %q: %w", dir, err)
		}
	}

	// Write project.json on first run.
	manifestPath := filepath.Join(projectPath, "project.json")
	projectName := filepath.Base(projectPath)
	if _, statErr := os.Stat(manifestPath); os.IsNotExist(statErr) {
		proj := models.Project{Name: projectName, CreatedAt: time.Now().Unix()}
		data, _ := json.MarshalIndent(proj, "", "  ")
		if err := os.WriteFile(manifestPath, data, 0644); err != nil {
			log.Printf("loadProject: warn: write project.json: %v", err)
		}
	}

	// Read project name from manifest.
	var projManifest struct {
		Name      string `json:"name"`
		CreatedAt int64  `json:"createdAt"`
	}
	if data, err := os.ReadFile(manifestPath); err == nil {
		_ = json.Unmarshal(data, &projManifest)
	}
	if projManifest.Name == "" {
		projManifest.Name = projectName
	}

	a.project = &models.Project{
		Name:      projManifest.Name,
		CreatedAt: projManifest.CreatedAt,
		Dir:       projectPath,
		ScenesDir: scenesDir,
		MirrorDir: mirrorDir,
		EngineDir: engineDir,
	}

	// Initialize SQLite.
	dbPath := filepath.Join(engineDir, "project.db")
	database, err := db.Initialize(dbPath)
	if err != nil {
		return fmt.Errorf("loadProject: db: %w", err)
	}
	a.database = database

	// Wire services.
	a.cache = services.NewCacheService(database)
	a.file = services.NewFileService(projectPath)
	a.events = services.NewEventService(a.mainWindow)
	a.mirror = services.NewMirrorService(mirrorDir)
	a.analysis = services.NewAnalysisService(a.cache, a.events)
	if a.llm != nil {
		a.llm.UpdateConfig(a.settings)
	}

	// Reset per-scene analysis tracking.
	a.waMu.Lock()
	a.wordCountAtLastAnalysis = make(map[string]int)
	a.waMu.Unlock()

	// Start background analysis goroutine with fresh context.
	a.ctx, a.cancel = context.WithCancel(context.Background())
	a.analysis.Start(a.ctx)

	// Sync disk ↔ DB.
	a.syncOnStartup()

	// Persist last project path.
	a.settings.LastProjectPath = projectPath
	a.saveSettings()

	log.Printf("loadProject: loaded %q", a.project.Name)
	return nil
}

func (a *App) OnShutdown() error {
	if a.cancel != nil {
		a.cancel()
	}
	if a.database != nil {
		_ = a.database.Close()
	}
	return nil
}

func (a *App) syncOnStartup() {
	files, err := a.file.ListSceneFiles(a.project.ScenesDir)
	if err != nil {
		log.Printf("sync: list: %v", err)
		return
	}
	diskPaths := make(map[string]bool)
	for _, abs := range files {
		rel := filepath.Base(abs)
		diskPaths[rel] = true
		exists, err := a.cache.FilePathExists(rel)
		if err != nil {
			continue
		}
		if !exists {
			maxIdx, _ := a.cache.GetMaxOrderIndex()
			scene := models.Scene{
				ID:           uuid.New().String(),
				Title:        strings.TrimSuffix(rel, ".md"),
				FilePath:     rel,
				OrderIndex:   maxIdx + 1,
				LastModified: time.Now().Unix(),
			}
			if err := a.cache.CreateScene(scene); err != nil {
				log.Printf("sync: register %q: %v", rel, err)
			}
		}
	}
	allScenes, _ := a.cache.GetAllScenes()
	for _, sc := range allScenes {
		if !diskPaths[sc.FilePath] {
			_ = a.cache.DeleteScene(sc.ID)
		}
	}
}

// ─── Project IPC ─────────────────────────────────────────────────────────────

// GetProjects scans ~/Documents/StoryEngine/ for project directories.
func (a *App) GetProjects() ([]models.ProjectInfo, error) {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, "Documents", "StoryEngine")
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("GetProjects: read dir: %w", err)
	}
	var projects []models.ProjectInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		manifestPath := filepath.Join(root, e.Name(), "project.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue // not a project directory
		}
		var proj struct {
			Name      string `json:"name"`
			CreatedAt int64  `json:"createdAt"`
		}
		if err := json.Unmarshal(data, &proj); err != nil {
			continue
		}
		projects = append(projects, models.ProjectInfo{
			Name:      proj.Name,
			Path:      filepath.Join(root, e.Name()),
			CreatedAt: proj.CreatedAt,
		})
	}
	return projects, nil
}

// CreateProject creates a new project directory and loads it.
func (a *App) CreateProject(name string) (models.ProjectInfo, error) {
	home, _ := os.UserHomeDir()
	slug := toSlug(name)
	if slug == "" {
		return models.ProjectInfo{}, fmt.Errorf("CreateProject: invalid name")
	}
	projectPath := filepath.Join(home, "Documents", "StoryEngine", slug)
	if dirExists(projectPath) {
		return models.ProjectInfo{}, fmt.Errorf("CreateProject: project %q already exists", slug)
	}

	// Write manifest before loadProject so the name is correct.
	if err := os.MkdirAll(projectPath, 0755); err != nil {
		return models.ProjectInfo{}, fmt.Errorf("CreateProject: mkdir: %w", err)
	}
	proj := struct {
		Name      string `json:"name"`
		CreatedAt int64  `json:"createdAt"`
	}{Name: name, CreatedAt: time.Now().Unix()}
	data, _ := json.MarshalIndent(proj, "", "  ")
	_ = os.WriteFile(filepath.Join(projectPath, "project.json"), data, 0644)

	if err := a.loadProject(projectPath); err != nil {
		return models.ProjectInfo{}, fmt.Errorf("CreateProject: load: %w", err)
	}
	return models.ProjectInfo{
		Name:      name,
		Path:      projectPath,
		CreatedAt: proj.CreatedAt,
	}, nil
}

// SwitchProject switches to an existing project and returns its scene list.
func (a *App) SwitchProject(projectPath string) ([]models.Scene, error) {
	if !dirExists(projectPath) {
		return nil, fmt.Errorf("SwitchProject: path does not exist: %q", projectPath)
	}
	if err := a.loadProject(projectPath); err != nil {
		return nil, fmt.Errorf("SwitchProject: %w", err)
	}
	return a.cache.GetAllScenes()
}

// GetCurrentProject returns metadata about the loaded project.
func (a *App) GetCurrentProject() models.ProjectInfo {
	if a.project == nil {
		return models.ProjectInfo{}
	}
	return models.ProjectInfo{
		Name:      a.project.Name,
		Path:      a.project.Dir,
		CreatedAt: a.project.CreatedAt,
	}
}

// GetSettings returns the current app settings.
func (a *App) GetSettings() (models.AppSettings, error) {
	return a.settings, nil
}

// SaveSettings persists settings and hot-swaps the LLM configuration.
func (a *App) SaveSettings(settings models.AppSettings) error {
	settings.LastProjectPath = a.settings.LastProjectPath
	a.settings = settings
	a.llm.UpdateConfig(settings)
	return a.saveSettings()
}

func (a *App) saveSettings() error {
	data, err := json.MarshalIndent(a.settings, "", "  ")
	if err != nil {
		return fmt.Errorf("saveSettings: marshal: %w", err)
	}
	if err := os.WriteFile(a.settingsPath, data, 0644); err != nil {
		return fmt.Errorf("saveSettings: write: %w", err)
	}
	return nil
}

// ─── Scene IPC ────────────────────────────────────────────────────────────────

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

// SaveSceneContent writes the scene to disk, updates word count, and triggers
// analysis. If the scene's word count has grown by ≥100 since the last LLM
// analysis, a background scene summary regeneration is triggered automatically.
func (a *App) SaveSceneContent(id string, content string) error {
	scene, err := a.cache.GetScene(id)
	if err != nil {
		return fmt.Errorf("SaveSceneContent: not found: %w", err)
	}
	if err := a.file.WriteScene(filepath.Join(a.project.ScenesDir, scene.FilePath), content); err != nil {
		return fmt.Errorf("SaveSceneContent: write: %w", err)
	}
	wc := countWords(content)
	if err := a.cache.UpdateWordCount(id, wc); err != nil {
		log.Printf("warn: word count update for %s: %v", id, err)
	}

	// Submit fast background entity detection.
	a.analysis.Submit(services.AnalysisJob{SceneID: id, Content: content})

	// Check 100-word threshold for automatic scene summary.
	a.waMu.Lock()
	lastWC := a.wordCountAtLastAnalysis[id]
	shouldAnalyze := wc-lastWC >= 100
	if shouldAnalyze {
		a.wordCountAtLastAnalysis[id] = wc
	}
	a.waMu.Unlock()

	if shouldAnalyze {
		go func() {
			if err := a.generateSceneSummary(id, scene.Title, content); err != nil {
				log.Printf("auto-analyze scene %s: %v", id, err)
			}
		}()
	}

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
	_ = os.Remove(filepath.Join(a.project.ScenesDir, scene.FilePath))
	return nil
}

// BatchReorderScenes replaces the old ReorderScene. Takes the full ordered slice
// of scene IDs and assigns sequential order indices in a single transaction.
func (a *App) BatchReorderScenes(orderedIDs []string) error {
	return a.cache.BatchReorderScenes(orderedIDs)
}

func (a *App) SaveCursorState(id string, cursorPos int, scrollTop float64) error {
	return a.cache.UpdateCursorState(id, cursorPos, scrollTop)
}

// ─── Mirror IPC ───────────────────────────────────────────────────────────────

// GetCharacters returns the project-wide character roster from mirror/characters.json.
func (a *App) GetCharacters() (map[string]models.CharacterProfile, error) {
	return a.mirror.GetCharacters()
}

// GetSceneInsights returns the stored summary for a scene from mirror/scenes/<id>.json.
func (a *App) GetSceneInsights(sceneID string) (models.SceneInsights, error) {
	summary, err := a.mirror.GetSceneSummary(sceneID)
	if err != nil {
		return models.SceneInsights{SceneID: sceneID}, nil
	}
	return models.SceneInsights{
		SceneID: sceneID,
		Summary: summary.Summary,
	}, nil
}

// AnalyzeProject calls the LLM with excerpts from all scenes to build/update
// the project-wide character roster. This is the 🧠 button at the project level.
func (a *App) AnalyzeProject() error {
	allScenes, err := a.cache.GetAllScenes()
	if err != nil {
		return fmt.Errorf("AnalyzeProject: get scenes: %w", err)
	}
	if len(allScenes) == 0 {
		return fmt.Errorf("AnalyzeProject: no scenes to analyze")
	}

	// Build scene text map.
	sceneTexts := make(map[string]string, len(allScenes))
	for _, sc := range allScenes {
		content, err := a.file.ReadScene(filepath.Join(a.project.ScenesDir, sc.FilePath))
		if err != nil {
			continue
		}
		sceneTexts[sc.Title] = content
	}

	result, err := a.llm.AnalyzeCharacters(a.project.Name, sceneTexts)
	if err != nil {
		return fmt.Errorf("AnalyzeProject: LLM: %w", err)
	}

	// Build entity→sceneIDs map from the DB.
	existingRoster, _ := a.mirror.GetCharacters()
	if existingRoster == nil {
		existingRoster = make(map[string]models.CharacterProfile)
	}

	for _, ch := range result.Characters {
		if ch.Name == "" {
			continue
		}
		sceneIDs, _ := a.cache.GetEntitySceneIDs(ch.Name)
		existingRoster[ch.Name] = models.CharacterProfile{
			Name:        ch.Name,
			Description: ch.Description,
			AppearsIn:   sceneIDs,
			UpdatedAt:   time.Now().Unix(),
		}
	}

	if err := a.mirror.SaveCharacters(existingRoster); err != nil {
		return fmt.Errorf("AnalyzeProject: save: %w", err)
	}

	// Convert for event payload.
	payload := make(map[string]interface{}, len(existingRoster))
	for k, v := range existingRoster {
		payload[k] = v
	}
	a.events.EmitCharactersUpdated(payload)
	return nil
}

// AnalyzeScene generates or regenerates the summary for a specific scene.
func (a *App) AnalyzeScene(sceneID string) error {
	scene, err := a.cache.GetScene(sceneID)
	if err != nil {
		return fmt.Errorf("AnalyzeScene: %w", err)
	}
	content, err := a.file.ReadScene(filepath.Join(a.project.ScenesDir, scene.FilePath))
	if err != nil {
		return fmt.Errorf("AnalyzeScene: read: %w", err)
	}
	if strings.TrimSpace(content) == "" {
		return fmt.Errorf("AnalyzeScene: scene is empty")
	}
	return a.generateSceneSummary(sceneID, scene.Title, content)
}

func (a *App) generateSceneSummary(sceneID, title, content string) error {
	summary, err := a.llm.SummarizeScene(title, content)
	if err != nil {
		return fmt.Errorf("generateSceneSummary: LLM: %w", err)
	}

	if err := a.mirror.SaveSceneSummary(models.SceneSummary{
		SceneID:   sceneID,
		Summary:   summary,
		UpdatedAt: time.Now().Unix(),
	}); err != nil {
		log.Printf("generateSceneSummary: save: %v", err)
	}

	a.events.EmitSceneSummaryUpdated(sceneID, summary)

	// Update threshold tracker.
	a.waMu.Lock()
	if scene, err := a.cache.GetScene(sceneID); err == nil {
		a.wordCountAtLastAnalysis[sceneID] = scene.WordCount
	}
	a.waMu.Unlock()

	return nil
}

// ─── Chat IPC ─────────────────────────────────────────────────────────────────

// AskQuestion answers a question about the project using keyword-ranked scene context.
func (a *App) AskQuestion(question string) (models.ChatResponse, error) {
	if strings.TrimSpace(question) == "" {
		return models.ChatResponse{}, fmt.Errorf("AskQuestion: empty question")
	}

	allScenes, err := a.cache.GetAllScenes()
	if err != nil {
		return models.ChatResponse{}, fmt.Errorf("AskQuestion: get scenes: %w", err)
	}

	// Rank scenes by keyword overlap with the question.
	ranked := rankScenesForQuestion(question, allScenes, a)

	// Build context from top-5 ranked scenes.
	type sceneCtx struct {
		Title   string
		Content string
	}
	var context []sceneCtx
	for _, rs := range ranked {
		content, err := a.file.ReadScene(filepath.Join(a.project.ScenesDir, rs.FilePath))
		if err != nil {
			continue
		}
		context = append(context, sceneCtx{Title: rs.Title, Content: content})
	}

	// Get last N chat messages for conversational continuity.
	history, _ := a.mirror.GetChatHistory()

	// Convert context for LLM.
	llmCtx := make([]struct{ Title, Content string }, len(context))
	for i, c := range context {
		llmCtx[i] = struct{ Title, Content string }{c.Title, c.Content}
	}

	rawAnswer, err := a.llm.AskQuestion(a.project.Name, question, llmCtx, history)
	if err != nil {
		return models.ChatResponse{}, fmt.Errorf("AskQuestion: LLM: %w", err)
	}

	answer, sources := services.ParseSourcesFromResponse(rawAnswer, allScenes)

	// Store in chat history.
	userMsg := models.ChatMessage{Role: "user", Content: question}
	assistantMsg := models.ChatMessage{Role: "assistant", Content: answer, Sources: sources}
	_ = a.mirror.AppendChatMessages(userMsg, assistantMsg)

	return models.ChatResponse{Answer: answer, Sources: sources}, nil
}

// GetChatHistory returns the project chat history.
func (a *App) GetChatHistory() ([]models.ChatMessage, error) {
	return a.mirror.GetChatHistory()
}

// ClearChat clears the project chat history.
func (a *App) ClearChat() error {
	return a.mirror.ClearChat()
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
	return "project"
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// rankScenesForQuestion scores scenes by keyword overlap with the question.
// Returns at most 5 scenes, ordered by score descending.
type rankedScene struct {
	models.Scene
	Score int
}

func rankScenesForQuestion(question string, scenes []models.Scene, a *App) []models.Scene {
	words := strings.Fields(strings.ToLower(question))

	var scored []rankedScene
	for _, sc := range scenes {
		lower := strings.ToLower(sc.Title)
		score := 0
		for _, w := range words {
			if len(w) > 3 {
				score += strings.Count(lower, w) * 3 // title matches weighted higher
			}
		}
		// Read a snippet for scoring.
		content, err := a.file.ReadScene(filepath.Join(a.project.ScenesDir, sc.FilePath))
		if err == nil {
			contentLower := strings.ToLower(content)
			for _, w := range words {
				if len(w) > 3 {
					score += strings.Count(contentLower, w)
				}
			}
		}
		scored = append(scored, rankedScene{sc, score})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	max := 5
	if len(scored) < max {
		max = len(scored)
	}
	result := make([]models.Scene, max)
	for i := range result {
		result[i] = scored[i].Scene
	}
	return result
}
