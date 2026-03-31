package models

// Scene represents a single writing scene.
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

// CharacterInteraction describes a detected exchange between characters.
// Summary is a prose-style margin note: "Elena and Marcus — heated argument."
type CharacterInteraction struct {
	Characters []string `json:"characters"`
	Tone       string   `json:"tone"`    // tense | warm | urgent | quiet | neutral
	Summary    string   `json:"summary"` // 6–12 words, written as a margin note
}

// MirrorPayload is the full analysis result, whether from rule-based or LLM analysis.
// Source is "rule" for background analysis and "llm" for LLM analysis.
// When source is "rule", Interactions will be empty and SceneTone will be "".
type MirrorPayload struct {
	SceneID      string                 `json:"sceneId"`
	Entities     []string               `json:"entities"`
	Interactions []CharacterInteraction `json:"interactions"`
	SceneTone    string                 `json:"sceneTone"`
	Source       string                 `json:"source"`
}

// SceneInsights is returned by GetInsights when loading a scene.
// InteractionsJSON is a raw JSON string to avoid re-encoding complexity at the IPC boundary.
type SceneInsights struct {
	SceneID          string   `json:"sceneId"`
	Entities         []string `json:"entities"`
	InteractionsJSON string   `json:"interactionsJSON"`
	SceneTone        string   `json:"sceneTone"`
	Source           string   `json:"source"`
	WordCount        int      `json:"wordCount"`
}

// AppSettings stores LLM configuration. Persisted to ~/Documents/StoryEngine/settings.json.
// APIKey is optional — leave empty for local models (Ollama, LM Studio).
type AppSettings struct {
	LLMEndpoint string `json:"llmEndpoint"` // e.g. "http://localhost:11434/v1"
	LLMModel    string `json:"llmModel"`    // e.g. "llama3.2:3b"
	LLMAPIKey   string `json:"llmApiKey"`   // empty for local; required for OpenAI/Groq
}

// DefaultSettings returns sane defaults pointing at a local Ollama instance.
func DefaultSettings() AppSettings {
	return AppSettings{
		LLMEndpoint: "http://localhost:11434/v1",
		LLMModel:    "llama3.2:3b",
		LLMAPIKey:   "",
	}
}

// Project is the runtime representation of project.json.
type Project struct {
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	ScenesDir string `json:"-"`
	EngineDir string `json:"-"`
}
