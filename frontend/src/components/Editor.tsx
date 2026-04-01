import { useEffect, useRef, useState } from 'react'
import { useEditor } from '../hooks/useEditor'
import { useSceneStore } from '../state/sceneStore'
import { useEditorStore } from '../state/editorStore'
import { RenameScene } from '../../bindings/story-engine/app'

export function Editor() {
  const { containerRef, loadScene, saveCursorState } = useEditor()
  const { scenes, activeSceneId, updateScene } = useSceneStore()
  const prevSceneIdRef = useRef<string | null>(null)
  const saveState = useEditorStore((s) => s.saveState)

  const activeScene = scenes.find((s) => s.id === activeSceneId) ?? null

  // Inline rename state for the status bar title.
  const [renaming, setRenaming] = useState(false)
  const [renameValue, setRenameValue] = useState('')
  const renameInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (renaming && renameInputRef.current) {
      renameInputRef.current.focus()
      renameInputRef.current.select()
    }
  }, [renaming])

  // Load scene when active scene changes.
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

  const handleTitleDoubleClick = () => {
    if (!activeScene) return
    setRenameValue(activeScene.title)
    setRenaming(true)
  }

  const commitRename = async () => {
    const trimmed = renameValue.trim()
    if (trimmed && activeScene && trimmed !== activeScene.title) {
      try {
        await RenameScene(activeScene.id, trimmed)
        updateScene(activeScene.id, { title: trimmed })
      } catch (err) {
        console.error('[Story Engine] RenameScene failed:', err)
      }
    }
    setRenaming(false)
  }

  return (
    <div className="flex flex-col flex-1 h-full bg-stone-base select-none overflow-hidden">
      {/* Status bar */}
      <div className="flex items-center justify-between px-4 py-2 border-b border-stone-border bg-stone-surface shrink-0">
        {/* Scene title — double-click to rename */}
        <div className="flex-1 min-w-0 mr-4">
          {renaming ? (
            <input
              ref={renameInputRef}
              value={renameValue}
              onChange={(e) => setRenameValue(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') commitRename()
                if (e.key === 'Escape') setRenaming(false)
              }}
              onBlur={commitRename}
              className="bg-transparent text-xs font-ui text-stone-text outline-none border-b border-stone-muted w-full"
              style={{ userSelect: 'text' }}
            />
          ) : (
            <span
              className="text-xs font-ui text-stone-subtle truncate block cursor-default"
              onDoubleClick={handleTitleDoubleClick}
              title={activeScene ? 'Double-click to rename' : ''}
            >
              {activeScene?.title ?? ''}
            </span>
          )}
        </div>

        {/* Save state + word count */}
        <div className="flex items-center gap-3 shrink-0">
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

      {/* Editor surface — always mounted */}
      <div className="flex-1 relative overflow-hidden">
        <div
          ref={containerRef}
          className="absolute inset-0"
          style={{ userSelect: 'text' }}
        />
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