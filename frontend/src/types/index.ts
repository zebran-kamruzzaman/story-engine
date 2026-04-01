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

export interface ProjectInfo {
  name: string
  path: string
  createdAt: number
}

export interface CharacterProfile {
  name: string
  description: string
  appearsIn: string[]
  updatedAt: number
}

export interface SceneInsights {
  sceneId: string
  summary: string
}

export interface ChatMessage {
  role: 'user' | 'assistant' | string;
  content: string
  sources?: SceneSource[]
}

export interface SceneSource {
  sceneId: string
  title: string
  score: number
}

export interface ChatResponse {
  answer: string
  sources: SceneSource[]
}

export interface AppSettings {
  llmEndpoint: string
  llmModel: string
  llmApiKey: string
}

// Event payloads — fired as DOM CustomEvents from Go via ExecJS.

export interface EntitiesUpdatedPayload {
  sceneId: string
  entities: string[]
}

export interface CharactersUpdatedPayload {
  [name: string]: CharacterProfile
}

export interface SceneSummaryUpdatedPayload {
  sceneId: string
  summary: string
}