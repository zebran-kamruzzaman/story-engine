import { create } from 'zustand'
import type { CharacterProfile, ChatMessage } from '../types'

// sceneSummaries maps sceneId → summary text.
// sceneEntities maps sceneId → entity names detected by rule-based analysis.
interface MirrorStore {
  // Project-wide character roster (LLM-generated)
  characters: Record<string, CharacterProfile>

  // Per-scene entities detected by fast rule-based analysis (updates on every save)
  sceneEntities: Record<string, string[]>

  // Per-scene LLM summaries
  sceneSummaries: Record<string, string>

  // Chat history for the current project
  chatHistory: ChatMessage[]

  // LLM analysis loading state
  isAnalyzingProject: boolean
  isAnalyzingScene: boolean
  analysisError: string | null

  setCharacters: (roster: Record<string, CharacterProfile>) => void
  updateCharacter: (name: string, profile: CharacterProfile) => void
  setSceneEntities: (sceneId: string, entities: string[]) => void
  setSceneSummary: (sceneId: string, summary: string) => void
  setChatHistory: (history: ChatMessage[]) => void
  appendChatMessage: (msg: ChatMessage) => void
  setAnalyzingProject: (v: boolean) => void
  setAnalyzingScene: (v: boolean) => void
  setAnalysisError: (msg: string | null) => void
  clearError: () => void
}

export const useMirrorStore = create<MirrorStore>((set) => ({
  characters: {},
  sceneEntities: {},
  sceneSummaries: {},
  chatHistory: [],
  isAnalyzingProject: false,
  isAnalyzingScene: false,
  analysisError: null,

  setCharacters: (characters) => set({ characters }),
  updateCharacter: (name, profile) =>
    set((s) => ({ characters: { ...s.characters, [name]: profile } })),
  setSceneEntities: (sceneId, entities) =>
    set((s) => ({ sceneEntities: { ...s.sceneEntities, [sceneId]: entities } })),
  setSceneSummary: (sceneId, summary) =>
    set((s) => ({ sceneSummaries: { ...s.sceneSummaries, [sceneId]: summary } })),
  setChatHistory: (chatHistory) => set({ chatHistory }),
  appendChatMessage: (msg) =>
    set((s) => ({ chatHistory: [...s.chatHistory, msg] })),
  setAnalyzingProject: (isAnalyzingProject) => set({ isAnalyzingProject }),
  setAnalyzingScene: (isAnalyzingScene) => set({ isAnalyzingScene }),
  setAnalysisError: (analysisError) =>
    set({ analysisError, isAnalyzingProject: false, isAnalyzingScene: false }),
  clearError: () => set({ analysisError: null }),
}))