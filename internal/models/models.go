package models

// Scene is a cache record. FilePath points to the authoritative .md file.
type Scene struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	FilePath       string  `json:"filePath"`
	OrderIndex     int     `json:"orderIndex"`
	WordCount      int     `json:"wordCount"`
	LastModified   int64   `json:"lastModified"`
	CursorPosition int     `json:"cursorPosition"`
	ScrollTop      float64 `json:"scrollTop"`
}

// ProjectInfo is returned by GetProjects and GetCurrentProject.
type ProjectInfo struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	CreatedAt int64  `json:"createdAt"`
}

// Project is the runtime project state. Fields without json tags are runtime-only.
type Project struct {
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	Dir       string `json:"-"` // absolute path to project root
	ScenesDir string `json:"-"`
	MirrorDir string `json:"-"`
	EngineDir string `json:"-"`
}

// CharacterProfile is one entry in the project-wide character roster.
type CharacterProfile struct {
	Name        string   `json:"name"`
	Description string   `json:"description"` // LLM-generated
	AppearsIn   []string `json:"appearsIn"`   // scene IDs
	UpdatedAt   int64    `json:"updatedAt"`
}

// SceneSummary is the LLM-generated summary for one scene.
type SceneSummary struct {
	SceneID   string `json:"sceneId"`
	Summary   string `json:"summary"`
	UpdatedAt int64  `json:"updatedAt"`
}

// ChatMessage is one turn in the project chat history.
type ChatMessage struct {
	Role    string        `json:"role"` // "user" | "assistant"
	Content string        `json:"content"`
	Sources []SceneSource `json:"sources,omitempty"` // populated for assistant messages
}

// SceneSource is a ranked scene reference in a chat response.
type SceneSource struct {
	SceneID string `json:"sceneId"`
	Title   string `json:"title"`
	Score   int    `json:"score"`
}

// ChatResponse is returned by the AskQuestion IPC method.
type ChatResponse struct {
	Answer  string        `json:"answer"`
	Sources []SceneSource `json:"sources"`
}

// AppSettings is persisted to ~/Documents/StoryEngine/app.json.
// APIKey is empty for local models (Ollama, LM Studio).
type AppSettings struct {
	LLMEndpoint     string `json:"llmEndpoint"`
	LLMModel        string `json:"llmModel"`
	LLMAPIKey       string `json:"llmApiKey"`
	LastProjectPath string `json:"lastProjectPath"`
}

// DefaultSettings returns sane defaults pointing at a local Ollama instance.
func DefaultSettings() AppSettings {
	return AppSettings{
		LLMEndpoint: "http://localhost:11434/v1",
		LLMModel:    "llama3.2:3b",
		LLMAPIKey:   "",
	}
}

// SceneInsights is returned by GetSceneInsights for the Mirror panel on scene switch.
type SceneInsights struct {
	SceneID string `json:"sceneId"`
	Summary string `json:"summary"` // may be empty if not yet generated
}
