import { create } from 'zustand'
import type { MirrorPayload } from '../types'

interface InsightsStore {
  mirror: MirrorPayload | null
  isAnalyzing: boolean
  analysisError: string | null
  setMirror: (payload: MirrorPayload) => void
  setAnalyzing: (v: boolean) => void
  setAnalysisError: (msg: string | null) => void
}

export const useInsightsStore = create<InsightsStore>((set) => ({
  mirror: null,
  isAnalyzing: false,
  analysisError: null,
  setMirror: (mirror) => set({ mirror, analysisError: null }),
  setAnalyzing: (isAnalyzing) => set({ isAnalyzing }),
  setAnalysisError: (analysisError) => set({ analysisError, isAnalyzing: false }),
}))