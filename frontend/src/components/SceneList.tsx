import { useState, useRef, useEffect } from 'react'
import type { MouseEvent } from 'react'
import { useSceneStore } from '../state/sceneStore'
import {
  CreateScene,
  DeleteScene,
  RenameScene,
  ReorderScene,
} from '../../bindings/story-engine/app'
import type { Scene } from '../types'

export function SceneList({
  onSceneSelect,
}: {
  onSceneSelect: (scene: Scene) => void
}) {
  const { scenes, activeSceneId, addScene, removeScene, updateScene } =
    useSceneStore()

  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const renameInputRef = useRef<HTMLInputElement>(null)

  // Total word count across all scenes
  const totalWords = scenes.reduce((sum, s) => sum + s.wordCount, 0)

  // Focus the rename input when it appears
  useEffect(() => {
    if (renamingId && renameInputRef.current) {
      renameInputRef.current.focus()
      renameInputRef.current.select()
    }
  }, [renamingId])

  const handleCreate = async () => {
    try {
      const scene = await CreateScene('Untitled')
      addScene(scene)
      // New scene enters rename mode immediately
      setRenamingId(scene.id)
      setRenameValue(scene.title)
    } catch (err) {
      console.error('[Story Engine] CreateScene failed:', err)
    }
  }

  const handleDelete = async (e: MouseEvent, id: string) => {
    e.stopPropagation()
    try {
      await DeleteScene(id)
      removeScene(id)
    } catch (err) {
      console.error('[Story Engine] DeleteScene failed:', err)
    }
  }

  const handleMoveUp = async (e: MouseEvent, scene: Scene) => {
    e.stopPropagation()
    const newIndex = Math.max(0, scene.orderIndex - 1)
    try {
      await ReorderScene(scene.id, newIndex)
      updateScene(scene.id, { orderIndex: newIndex })
      const displaced = scenes.find(
        (s) => s.id !== scene.id && s.orderIndex === newIndex
      )
      if (displaced) updateScene(displaced.id, { orderIndex: scene.orderIndex })
    } catch (err) {
      console.error('[Story Engine] ReorderScene failed:', err)
    }
  }

  const handleMoveDown = async (e: MouseEvent, scene: Scene) => {
    e.stopPropagation()
    const newIndex = scene.orderIndex + 1
    try {
      await ReorderScene(scene.id, newIndex)
      updateScene(scene.id, { orderIndex: newIndex })
      const displaced = scenes.find(
        (s) => s.id !== scene.id && s.orderIndex === newIndex
      )
      if (displaced) updateScene(displaced.id, { orderIndex: scene.orderIndex })
    } catch (err) {
      console.error('[Story Engine] ReorderScene failed:', err)
    }
  }

  const startRename = (scene: Scene) => {
    setRenamingId(scene.id)
    setRenameValue(scene.title)
  }

  const commitRename = async (id: string) => {
    const trimmed = renameValue.trim()
    if (trimmed) {
      try {
        await RenameScene(id, trimmed)
        updateScene(id, { title: trimmed })
      } catch (err) {
        console.error('[Story Engine] RenameScene failed:', err)
      }
    }
    setRenamingId(null)
  }

  const cancelRename = () => setRenamingId(null)

  const sortedScenes = [...scenes].sort((a, b) => a.orderIndex - b.orderIndex)

  return (
    <div className="flex flex-col h-full bg-stone-surface border-r border-stone-border select-none">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-3 border-b border-stone-border">
        <span className="text-xs tracking-widest uppercase text-stone-muted font-ui">
          Scenes
        </span>
        <button
          onClick={handleCreate}
          className="w-5 h-5 flex items-center justify-center text-stone-subtle hover:text-stone-text transition-colors"
          title="New scene"
        >
          +
        </button>
      </div>

      {/* Scene list */}
      <div className="flex-1 overflow-y-auto">
        {sortedScenes.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full px-4 text-center">
            <p className="text-stone-subtle text-xs font-ui">
              No scenes yet.{' '}
              <button
                onClick={handleCreate}
                className="text-stone-text underline"
              >
                Create your first scene
              </button>
            </p>
          </div>
        ) : (
          sortedScenes.map((scene) => {
            const isActive = scene.id === activeSceneId
            return (
              <div
                key={scene.id}
                onClick={() => onSceneSelect(scene)}
                className={`group relative flex flex-col px-3 py-2 cursor-pointer border-l-2 transition-colors ${
                  isActive
                    ? 'border-l-stone-text bg-stone-border'
                    : 'border-l-transparent hover:bg-stone-border/50'
                }`}
              >
                {renamingId === scene.id ? (
                  <input
                    ref={renameInputRef}
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') commitRename(scene.id)
                      if (e.key === 'Escape') cancelRename()
                    }}
                    onBlur={() => commitRename(scene.id)}
                    onClick={(e) => e.stopPropagation()}
                    className="bg-stone-border text-stone-text text-xs font-ui w-full outline-none border-b border-stone-muted"
                  />
                ) : (
                  <span
                    className="text-xs font-ui text-stone-text truncate"
                    onDoubleClick={(e) => {
                      e.stopPropagation()
                      startRename(scene)
                    }}
                  >
                    {scene.title}
                  </span>
                )}

                <span className="text-[10px] text-stone-subtle font-ui mt-0.5">
                  {scene.wordCount.toLocaleString()} words
                </span>

                {/* Action buttons — visible on hover */}
                <div className="absolute right-2 top-1/2 -translate-y-1/2 hidden group-hover:flex items-center gap-1">
                  <button
                    onClick={(e) => handleMoveUp(e, scene)}
                    className="text-stone-subtle hover:text-stone-text text-xs px-0.5"
                    title="Move up"
                  >
                    ↑
                  </button>
                  <button
                    onClick={(e) => handleMoveDown(e, scene)}
                    className="text-stone-subtle hover:text-stone-text text-xs px-0.5"
                    title="Move down"
                  >
                    ↓
                  </button>
                  <button
                    onClick={(e) => handleDelete(e, scene.id)}
                    className="text-stone-subtle hover:text-red-400 text-xs px-0.5"
                    title="Delete scene"
                  >
                    ×
                  </button>
                </div>
              </div>
            )
          })
        )}
      </div>

      {/* Footer: total word count */}
      <div className="px-3 py-2 border-t border-stone-border">
        <span className="text-[10px] text-stone-subtle font-ui">
          {totalWords.toLocaleString()} total words
        </span>
      </div>
    </div>
  )
}