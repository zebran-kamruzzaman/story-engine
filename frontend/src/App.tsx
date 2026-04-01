import { useEffect } from 'react'
import {
  GetScenes,
  GetCurrentProject,
  GetProjects,
  GetCharacters,
} from '../bindings/story-engine/app'
import { useSceneStore } from './state/sceneStore'
import { useProjectStore } from './state/projectStore'
import { useMirrorStore } from './state/mirrorStore'
import { MenuBar } from './components/MenuBar'
import { SceneList } from './components/SceneList'
import { Editor } from './components/Editor'
import { MirrorPanel } from './components/MirrorPanel'
import type {
  Scene,
  CharacterProfile,
  EntitiesUpdatedPayload,
  CharactersUpdatedPayload,
  SceneSummaryUpdatedPayload,
} from './types'

export default function App() {
  const { setScenes, setActiveSceneId } = useSceneStore()
  const { setCurrentProject, setProjects } = useProjectStore()
  const { setCharacters, setSceneEntities, setSceneSummary } = useMirrorStore()

  // Load initial data for the project that was open on backend startup.
  useEffect(() => {
    ;(async () => {
      try {
        const [scenes, project, projects, rawCharacters] = await Promise.all([
          GetScenes(),
          GetCurrentProject(),
          GetProjects(),
          GetCharacters(),
        ])
        const list = scenes ?? []
        setScenes(list)
        setCurrentProject(project)
        setProjects(projects ?? [])

        // GetCharacters() returns { [key: string]?: CharacterProfile } — values
        // may be undefined due to TypeScript's optional index signature on Go maps.
        // Cast to the store's expected type; undefined values are never present
        // in practice because Go serializes all map values.
        setCharacters(
          (rawCharacters ?? {}) as Record<string, CharacterProfile>
        )

        if (list.length > 0) setActiveSceneId(list[0].id)
      } catch (err) {
        console.error('[Story Engine] initial load failed:', err)
      }
    })()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // mirror:entities-updated — fired on every save by rule-based analysis
  useEffect(() => {
    const handler = (e: Event) => {
      const { sceneId, entities } = (e as CustomEvent<EntitiesUpdatedPayload>).detail
      setSceneEntities(sceneId, entities ?? [])
    }
    window.addEventListener('mirror:entities-updated', handler)
    return () => window.removeEventListener('mirror:entities-updated', handler)
  }, [setSceneEntities])

  // mirror:characters-updated — fired after AnalyzeProject() completes
  useEffect(() => {
    const handler = (e: Event) => {
      const roster = (e as CustomEvent<CharactersUpdatedPayload>).detail
      setCharacters((roster ?? {}) as Record<string, CharacterProfile>)
    }
    window.addEventListener('mirror:characters-updated', handler)
    return () => window.removeEventListener('mirror:characters-updated', handler)
  }, [setCharacters])

  // mirror:scene-summary-updated — fired after scene summary generation
  useEffect(() => {
    const handler = (e: Event) => {
      const { sceneId, summary } = (e as CustomEvent<SceneSummaryUpdatedPayload>).detail
      setSceneSummary(sceneId, summary)
    }
    window.addEventListener('mirror:scene-summary-updated', handler)
    return () => window.removeEventListener('mirror:scene-summary-updated', handler)
  }, [setSceneSummary])

  const handleSceneSelect = (scene: Scene) => setActiveSceneId(scene.id)

  return (
    <div className="flex flex-col h-screen w-screen bg-stone-base overflow-hidden">
      <MenuBar />
      <div className="flex flex-1 overflow-hidden">
        <div className="w-56 shrink-0">
          <SceneList onSceneSelect={handleSceneSelect} />
        </div>
        <Editor />
        <MirrorPanel />
      </div>
    </div>
  )
}