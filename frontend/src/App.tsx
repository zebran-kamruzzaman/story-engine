import { useEffect } from 'react'
import { GetScenes } from '../bindings/story-engine/app'
import { useSceneStore } from './state/sceneStore'
import { useInsightsStore } from './state/insightsStore'
import { SceneList } from './components/SceneList'
import { Editor } from './components/Editor'
import { InsightsPanel } from './components/InsightsPanel'
import type { MirrorPayload, Scene } from './types'

export default function App() {
  const { setScenes, setActiveSceneId } = useSceneStore()
  const setMirror = useInsightsStore((s) => s.setMirror)

  useEffect(() => {
    ;(async () => {
      try {
        const loaded = await GetScenes()
        const list = loaded ?? []
        setScenes(list)
        if (list.length > 0) setActiveSceneId(list[0].id)
      } catch (err) {
        console.error('[Story Engine] GetScenes failed:', err)
      }
    })()
  }, [setScenes, setActiveSceneId])

  // Listen for mirror updates from the Go backend.
  // Both rule-based (on save) and LLM (on brain icon click) fire this event.
  useEffect(() => {
    const handler = (e: Event) => {
      const payload = (e as CustomEvent<MirrorPayload>).detail
      setMirror(payload)
    }
    window.addEventListener('mirror:updated', handler)
    return () => window.removeEventListener('mirror:updated', handler)
  }, [setMirror])

  const handleSceneSelect = (scene: Scene) => setActiveSceneId(scene.id)

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