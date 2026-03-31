import { useEffect, useRef } from 'react'
import { useEditor } from '../hooks/useEditor'
import { useSceneStore } from '../state/sceneStore'
import { useEditorStore } from '../state/editorStore'

export function Editor() {
  const { containerRef, loadScene, saveCursorState } = useEditor()
  const { scenes, activeSceneId } = useSceneStore()
  const prevSceneIdRef = useRef<string | null>(null)

  const activeScene = scenes.find((s) => s.id === activeSceneId) ?? null
  const saveState = useEditorStore((s) => s.saveState)

  useEffect(() => {
    const prevId = prevSceneIdRef.current
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
        <div className="flex items-center gap-3">
          {saveState === 'saving' && (
            <span className="text-xs font-ui text-stone-subtle italic">Saving…</span>
          )}
          {saveState === 'saved' && (
            <span className="text-xs font-ui text-stone-muted">Saved</span>
          )}
          <span className="text-xs font-ui text-stone-subtle">
            {activeScene ? `${activeScene.wordCount.toLocaleString()} words` : ''}
          </span>
        </div>
      </div>

      {/* Editor area: always-mounted CodeMirror container + conditional overlay */}
      <div className="flex-1 relative overflow-hidden">
        {/*
          This div is ALWAYS in the DOM. That's the fix.
          useEditor's useEffect fires once on mount and finds containerRef.current
          non-null, so EditorView gets created immediately.
          Previously this was inside a conditional block — the effect ran before
          the div existed and returned early, leaving the editor permanently unmounted.
        */}
        <div
          ref={containerRef}
          className="absolute inset-0"
          style={{ userSelect: 'text' }}
        />

        {/* Empty state overlay — sits on top of the editor when no scene is open */}
        {!activeSceneId && (
          <div className="absolute inset-0 flex items-center justify-center bg-stone-base pointer-events-none">
            <p className="text-stone-subtle text-sm font-ui italic">
              No scene selected
            </p>
          </div>
        )}
      </div>
    </div>
  )
}