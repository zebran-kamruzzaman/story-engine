import { create } from 'zustand'

type SaveState = 'idle' | 'saving' | 'saved'

interface EditorStore {
  saveState: SaveState
  setSaveState: (state: SaveState) => void
}

export const useEditorStore = create<EditorStore>((set) => ({
  saveState: 'idle',
  setSaveState: (saveState) => set({ saveState }),
}))