package models

// Scene represents a single writing scene.
// It is a cache record — the FilePath field points to the authoritative .md file.
// Deleting this record does not delete the file.
type Scene struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	FilePath       string  `json:"filePath"` // relative to project ScenesDir
	OrderIndex     int     `json:"orderIndex"`
	WordCount      int     `json:"wordCount"`
	LastModified   int64   `json:"lastModified"`   // Unix timestamp
	CursorPosition int     `json:"cursorPosition"` // CodeMirror offset
	ScrollTop      float64 `json:"scrollTop"`      // pixels, from scrollDOM
}

// SceneInsights is the full insights payload returned by GetInsights.
// It combines entity data from SQLite with the current word count.
type SceneInsights struct {
	SceneID       string   `json:"sceneId"`
	Entities      []string `json:"entities"`      // candidate character names
	DialogueCount int      `json:"dialogueCount"` // estimated dialogue segments
	WordCount     int      `json:"wordCount"`
}

// InsightsPayload is the lightweight event payload emitted after analysis completes.
// It does not include WordCount because the frontend updates that from the save response.
type InsightsPayload struct {
	SceneID       string   `json:"sceneId"`
	Entities      []string `json:"entities"`
	DialogueCount int      `json:"dialogueCount"`
}

// Project is the runtime representation of project.json.
// ScenesDir and EngineDir are set at runtime and never serialized.
type Project struct {
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	ScenesDir string `json:"-"` // runtime only
	EngineDir string `json:"-"` // runtime only
}
