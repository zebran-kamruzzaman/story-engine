import { create } from 'zustand'
import type { InsightsPayload } from '../types'

interface InsightsStore {
  insights: InsightsPayload | null
  setInsights: (payload: InsightsPayload) => void
}

export const useInsightsStore = create<InsightsStore>((set) => ({
  insights: null,
  setInsights: (payload) => set({ insights: payload }),
}))