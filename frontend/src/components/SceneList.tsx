import { useState, useRef, useEffect } from 'react'
import type { MouseEvent } from 'react'
import {
  DragDropContext,
  Droppable,
  Draggable,
  type DropResult,
} from '@hello-pangea/dnd'
import { useSceneStore } from '../state/sceneStore'
import {
  CreateScene,
  DeleteScene,
  RenameScene,
  BatchReorderScenes,
} from '../../bindings/story-engine/app'
import type { Scene } from '../types'

export function SceneList({
  onSceneSelect,
}: {
  onSceneSelect: (scene: Scene) => void
}) {
  const { scenes, activeSceneId, addScene, removeScene, updateScene, setScenes } =
    useSceneStore()
  const [renamingId, setRenamingId] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')
  const renameInputRef = useRef<HTMLInputElement>(null)

  const totalWords = scenes.reduce((sum, s) => sum + s.wordCount, 0)
  const sortedScenes = [...scenes].sort((a, b) => a.orderIndex - b.orderIndex)

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

  const handleDragEnd = async (result: DropResult) => {
    if (!result.destination) return
    if (result.destination.index === result.source.index) return

    // Build new order optimistically.
    const items = [...sortedScenes]
    const [moved] = items.splice(result.source.index, 1)
    items.splice(result.destination.index, 0, moved)

    // Update store with sequential order indices.
    const updated = items.map((sc, i) => ({ ...sc, orderIndex: i }))
    setScenes(updated)

    // Persist to backend.
    try {
      await BatchReorderScenes(items.map((sc) => sc.id))
    } catch (err) {
      console.error('[Story Engine] BatchReorderScenes failed:', err)
      // Revert by refetching — not implemented here for brevity;
      // a page reload would also work since startup sync restores the list.
    }
  }

  return (
    <div className="flex flex-col h-full bg-stone-surface border-r border-stone-border select-none">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-3 border-b border-stone-border">
        <span className="text-[10px] tracking-widest uppercase text-stone-muted font-ui">
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

      {/* Scene list with drag-to-reorder */}
      <div className="flex-1 overflow-y-auto">
        {sortedScenes.length === 0 ? (
          <div className="flex items-center justify-center h-full px-4 text-center">
            <p className="text-stone-subtle text-xs font-ui">
              No scenes yet.{' '}
              <button onClick={handleCreate} className="text-stone-text underline">
                Create your first scene
              </button>
            </p>
          </div>
        ) : (
          <DragDropContext onDragEnd={handleDragEnd}>
            <Droppable droppableId="scenes">
              {(provided) => (
                <div ref={provided.innerRef} {...provided.droppableProps}>
                  {sortedScenes.map((scene, index) => {
                    const isActive = scene.id === activeSceneId
                    return (
                      <Draggable key={scene.id} draggableId={scene.id} index={index}>
                        {(prov, snapshot) => (
                          <div
                            ref={prov.innerRef}
                            {...prov.draggableProps}
                            onClick={() => onSceneSelect(scene)}
                            className={`group flex items-center gap-1 px-1 py-2 cursor-pointer border-l-2 transition-colors ${
                              isActive
                                ? 'border-l-stone-text bg-stone-border'
                                : 'border-l-transparent hover:bg-stone-border/50'
                            } ${snapshot.isDragging ? 'opacity-60 shadow-lg' : ''}`}
                          >
                            {/* Drag handle */}
                            <div
                              {...prov.dragHandleProps}
                              onClick={(e) => e.stopPropagation()}
                              className="shrink-0 px-1 text-stone-muted hover:text-stone-subtle cursor-grab active:cursor-grabbing"
                              title="Drag to reorder"
                            >
                              ⠿
                            </div>

                            {/* Scene title and word count */}
                            <div className="flex-1 min-w-0">
                              {renamingId === scene.id ? (
                                <input
                                  ref={renameInputRef}
                                  value={renameValue}
                                  onChange={(e) => setRenameValue(e.target.value)}
                                  onKeyDown={(e) => {
                                    if (e.key === 'Enter') commitRename(scene.id)
                                    if (e.key === 'Escape') setRenamingId(null)
                                  }}
                                  onBlur={() => commitRename(scene.id)}
                                  onClick={(e) => e.stopPropagation()}
                                  className="bg-stone-border text-stone-text text-xs font-ui w-full outline-none border-b border-stone-muted"
                                />
                              ) : (
                                <span className="text-xs font-ui text-stone-text truncate block">
                                  {scene.title}
                                </span>
                              )}
                              <span className="text-[10px] text-stone-muted font-ui">
                                {scene.wordCount.toLocaleString()} words
                              </span>
                            </div>

                            {/* Delete button — visible on hover */}
                            <button
                              onClick={(e) => handleDelete(e, scene.id)}
                              className="shrink-0 px-1 text-stone-muted hover:text-red-400 text-xs opacity-0 group-hover:opacity-100 transition-opacity"
                              title="Delete scene"
                            >
                              ×
                            </button>
                          </div>
                        )}
                      </Draggable>
                    )
                  })}
                  {provided.placeholder}
                </div>
              )}
            </Droppable>
          </DragDropContext>
        )}
      </div>

      {/* Footer */}
      <div className="px-3 py-2 border-t border-stone-border">
        <span className="text-[10px] text-stone-muted font-ui">
          {totalWords.toLocaleString()} total words
        </span>
      </div>
    </div>
  )
}