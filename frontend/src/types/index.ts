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

export interface CharacterInteraction {
  characters: string[]
  tone: string    // tense | warm | urgent | quiet | neutral
  summary: string // prose margin note
}

// MirrorPayload is the live event payload from "mirror:updated" CustomEvent.
export interface MirrorPayload {
  sceneId: string
  entities: string[]
  interactions: CharacterInteraction[]
  sceneTone: string
  source: 'rule' | 'llm'
}

// SceneInsights is returned by GetInsights() when switching scenes.
export interface SceneInsights {
  sceneId: string
  entities: string[]
  interactionsJSON: string  // raw JSON; parse to CharacterInteraction[]
  sceneTone: string
  source: string
  wordCount: number
}

export interface AppSettings {
  llmEndpoint: string
  llmModel: string
  llmApiKey: string
}