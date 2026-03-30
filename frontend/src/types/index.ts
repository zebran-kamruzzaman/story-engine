// Scene represents one writing scene. Maps to models.Scene in Go.
export interface Scene {
  id: string
  title: string
  filePath: string
  orderIndex: number
  wordCount: number
  lastModified: number
  cursorPosition: number
  scrollTop: number
}

// SceneInsights is the full insights response from GetInsights(). Maps to models.SceneInsights.
export interface SceneInsights {
  sceneId: string
  entities: string[]
  dialogueCount: number
  wordCount: number
}

// InsightsPayload is the event payload pushed by the backend after analysis.
// Maps to models.InsightsPayload.
export interface InsightsPayload {
  sceneId: string
  entities: string[]
  dialogueCount: number
}