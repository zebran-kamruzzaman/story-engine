import { useEffect } from 'react'
import { GetScenes } from '../bindings/story-engine/app'
import { useSceneStore } from './state/sceneStore'
import { useInsightsStore } from './state/insightsStore'
import { SceneList } from './components/SceneList'
import { Editor } from './components/Editor'
import { InsightsPanel } from './components/InsightsPanel'
import type { InsightsPayload, Scene } from './types'

// NOTE: The GetScenes (and all other IPC calls) import path comes from the
// generated bindings folder. Wails v3 generates into frontend/bindings/ rather
// than frontend/wailsjs/. Check your actual generated path after wails3 dev
// and adjust these imports to match if needed.
// Common paths in alpha.74:
//   ../bindings/github.com/wailsapp/wails/v3/pkg/application  (template default)
//   ../../wailsjs/go/main/App  (older alpha style)

export default function App() {
  const { setScenes, setActiveSceneId } = useSceneStore()
  const setInsights = useInsightsStore((s) => s.setInsights)

  // Load initial scene list on mount.
  useEffect(() => {
    ;(async () => {
      try {
        const scenes = await GetScenes()
        setScenes(scenes ?? [])
      } catch (err) {
        console.error('[Story Engine] GetScenes failed:', err)
      }
    })()
  }, [setScenes])

  // Listen for insights events dispatched from Go via ExecJS.
  // We use a plain DOM CustomEvent instead of the Wails event API because
  // application.WailsEvent / application.CustomEvent keep changing between
  // alpha.74 releases. window.addEventListener is stable regardless.
  useEffect(() => {
    const handler = (e: Event) => {
      const payload = (e as CustomEvent<InsightsPayload>).detail
      setInsights(payload)
    }
    window.addEventListener('insights:updated', handler)
    return () => window.removeEventListener('insights:updated', handler)
  }, [setInsights])

  const handleSceneSelect = (scene: Scene) => {
    setActiveSceneId(scene.id)
  }

  return (
    <div className="flex h-screen w-screen bg-stone-base overflow-hidden">
      <div className="w-56 shrink-0">
        <SceneList onSceneSelect={handleSceneSelect} />
      </div>
      <Editor />
      <InsightsPanel />
    </div>
  )
}