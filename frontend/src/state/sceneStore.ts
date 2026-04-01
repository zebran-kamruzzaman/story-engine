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
  addScene: (scene) => set((s) => ({ scenes: [...s.scenes, scene] })),
  removeScene: (id) =>
    set((s) => ({
      scenes: s.scenes.filter((sc) => sc.id !== id),
      activeSceneId: s.activeSceneId === id ? null : s.activeSceneId,
    })),
  updateScene: (id, updates) =>
    set((s) => ({
      scenes: s.scenes.map((sc) => (sc.id === id ? { ...sc, ...updates } : sc)),
    })),
  setActiveSceneId: (id) => set({ activeSceneId: id }),
}))