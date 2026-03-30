import { useEffect, useRef } from 'react'
import { useEditor } from '../hooks/useEditor'
import { useSceneStore } from '../state/sceneStore'

export function Editor() {
  const { containerRef, loadScene, saveCursorState } = useEditor()
  const { scenes, activeSceneId } = useSceneStore()
  const prevSceneIdRef = useRef<string | null>(null)

  const activeScene = scenes.find((s) => s.id === activeSceneId) ?? null

  useEffect(() => {
    const prevId = prevSceneIdRef.current

    // Save cursor state of the scene we are leaving.
    if (prevId && prevId !== activeSceneId) {
      saveCursorState(prevId)
    }

    if (activeSceneId && activeScene) {
      loadScene(activeSceneId, activeScene.cursorPosition, activeScene.scrollTop)
    }

    prevSceneIdRef.current = activeSceneId ?? null
  }, [activeSceneId]) // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <div className="flex flex-col flex-1 h-full bg-stone-base select-none">
      {/* Status bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-stone-border bg-stone-surface shrink-0">
        <span className="text-xs font-ui text-stone-subtle">
          {activeScene?.title ?? ''}
        </span>
        <span className="text-xs font-ui text-stone-subtle">
          {activeScene ? `${activeScene.wordCount.toLocaleString()} words` : ''}
        </span>
      </div>

      {/* Editor surface */}
      {activeSceneId ? (
        <div
          ref={containerRef}
          className="flex-1 overflow-hidden"
          style={{ userSelect: 'text' }}
        />
      ) : (
        <div className="flex-1 flex items-center justify-center">
          <p className="text-stone-subtle text-sm font-ui italic">
            No scene selected
          </p>
        </div>
      )}
    </div>
  )
}