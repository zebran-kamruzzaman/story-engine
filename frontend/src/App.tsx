import { useEffect } from 'react'
import { GetScenes } from '../bindings/story-engine/app'
import { useSceneStore } from './state/sceneStore'
import { useInsightsStore } from './state/insightsStore'
import { SceneList } from './components/SceneList'
import { Editor } from './components/Editor'
import { InsightsPanel } from './components/InsightsPanel'
import type { InsightsPayload, Scene } from './types'

export default function App() {
  const { setScenes, setActiveSceneId } = useSceneStore()
  const setInsights = useInsightsStore((s) => s.setInsights)

  // Load initial scene list. Auto-select the first scene if one exists.
  useEffect(() => {
    ;(async () => {
      try {
        const loaded = await GetScenes()
        const list = loaded ?? []
        setScenes(list)
        if (list.length > 0) {
          setActiveSceneId(list[0].id)
        }
      } catch (err) {
        console.error('[Story Engine] GetScenes failed:', err)
      }
    })()
  }, [setScenes, setActiveSceneId])

  // Listen for insights events dispatched from Go via ExecJS.
  // event_service.go fires: window.dispatchEvent(new CustomEvent('insights:updated', {detail: ...}))
  // We must use window.addEventListener here — NOT Events.On from @wailsio/runtime —
  // because ExecJS fires a DOM CustomEvent, not a Wails runtime event.
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