import { create } from 'zustand'
import type { Scene } from '../types'

interface SceneStore {
  scenes: Scene[]
  activeSceneId: string | null
  setScenes: (scenes: Scene[]) => void
  addScene: (scene: Scene) => void
  removeScene: (id: string) => void
  updateScene: (id: string, updates: Partial<Scene>) => void
  setActiveSceneId: (id: string | null) => void
}

export const useSceneStore = create<SceneStore>((set) => ({
  scenes: [],
  activeSceneId: null,

  setScenes: (scenes) => set({ scenes }),

  addScene: (scene) =>
    set((state) => ({ scenes: [...state.scenes, scene] })),

  removeScene: (id) =>
    set((state) => ({
      scenes: state.scenes.filter((s) => s.id !== id),
      activeSceneId: state.activeSceneId === id ? null : state.activeSceneId,
    })),

  updateScene: (id, updates) =>
    set((state) => ({
      scenes: state.scenes.map((s) => (s.id === id ? { ...s, ...updates } : s)),
    })),

  setActiveSceneId: (id) => set({ activeSceneId: id }),
}))