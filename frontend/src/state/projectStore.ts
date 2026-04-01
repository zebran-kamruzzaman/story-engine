import { create } from 'zustand'
import type { ProjectInfo } from '../types'

interface ProjectStore {
  currentProject: ProjectInfo | null
  projects: ProjectInfo[]
  setCurrentProject: (p: ProjectInfo) => void
  setProjects: (list: ProjectInfo[]) => void
}

export const useProjectStore = create<ProjectStore>((set) => ({
  currentProject: null,
  projects: [],
  setCurrentProject: (currentProject) => set({ currentProject }),
  setProjects: (projects) => set({ projects }),
}))