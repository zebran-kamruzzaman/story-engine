import { useEffect, useRef, useCallback } from 'react'
import { EditorView, keymap } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { markdown } from '@codemirror/lang-markdown'
import { useSceneStore } from '../state/sceneStore'
import { useEditorStore } from '../state/editorStore'
import { useMirrorStore } from '../state/mirrorStore'
import {
  SaveSceneContent,
  SaveCursorState,
  GetSceneContent,
  GetSceneInsights,
} from '../../bindings/story-engine/app'

// ─── CodeMirror Stone Theme ────────────────────────────────────────────────

const stoneTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: '#1B1B1B',
      color: '#D1D1D1',
      height: '100%',
      fontFamily: 'Georgia, Cambria, "Times New Roman", serif',
      fontSize: '17px',
    },
    '.cm-content': {
      padding: '64px 0',
      maxWidth: '680px',
      margin: '0 auto',
      caretColor: '#D1D1D1',
      lineHeight: '1.85',
    },
    '.cm-line': { padding: '0 24px' },
    '.cm-cursor, .cm-dropCursor': {
      borderLeftColor: '#D1D1D1',
      borderLeftWidth: '1.5px',
    },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
      backgroundColor: '#2E2E2E !important',
    },
    '.cm-scroller': { overflowY: 'auto', fontFamily: 'inherit' },
    '.cm-gutters': { display: 'none' },
  },
  { dark: true }
)

// ─── useEditor Hook ─────────────────────────────────────────────────────────

export function useEditor() {
  const containerRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const currentSceneIdRef = useRef<string | null>(null)

  const updateScene = useSceneStore((s) => s.updateScene)
  const setSaveState = useEditorStore((s) => s.setSaveState)
  const setSceneSummary = useMirrorStore((s) => s.setSceneSummary)

  const saveCursorState = useCallback(async (id: string) => {
    const view = viewRef.current
    if (!view) return
    try {
      await SaveCursorState(id, view.state.selection.main.anchor, view.scrollDOM.scrollTop)
    } catch (err) {
      console.error('[Story Engine] SaveCursorState failed:', err)
    }
  }, [])

  const loadScene = useCallback(
    async (id: string, cursorPos: number, scrollTop: number) => {
      const view = viewRef.current
      if (!view) return

      if (debounceRef.current) {
        clearTimeout(debounceRef.current)
        debounceRef.current = null
      }
      currentSceneIdRef.current = id

      // Clear → fetch → insert → restore
      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: '' } })

      let content = ''
      try {
        content = await GetSceneContent(id)
      } catch (err) {
        console.error('[Story Engine] GetSceneContent failed:', err)
      }

      view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: content } })
      view.dispatch({ selection: { anchor: Math.min(cursorPos, view.state.doc.length) } })
      requestAnimationFrame(() => {
        if (viewRef.current) viewRef.current.scrollDOM.scrollTop = scrollTop
      })
      view.focus()

      // Load last-known scene summary for this scene.
      try {
        const insights = await GetSceneInsights(id)
        if (insights.summary) {
          setSceneSummary(id, insights.summary)
        }
      } catch (err) {
        console.error('[Story Engine] GetSceneInsights failed:', err)
      }
    },
    [setSceneSummary]
  )

  useEffect(() => {
    if (!containerRef.current) return

    const updateListener = EditorView.updateListener.of((update) => {
      if (!update.docChanged) return
      const id = currentSceneIdRef.current
      if (!id) return

      if (debounceRef.current) clearTimeout(debounceRef.current)
      setSaveState('saving')
      debounceRef.current = setTimeout(async () => {
        const content = update.state.doc.toString()
        try {
          await SaveSceneContent(id, content)
          const wordCount = content.trim() === '' ? 0 : content.trim().split(/\s+/).length
          updateScene(id, { wordCount })
          setSaveState('saved')
          setTimeout(() => setSaveState('idle'), 2000)
        } catch (err) {
          console.error('[Story Engine] SaveSceneContent failed:', err)
          setSaveState('idle')
        }
      }, 800)
    })

    const state = EditorState.create({
      doc: '',
      extensions: [
        history(),
        markdown(),
        EditorView.lineWrapping,
        stoneTheme,
        keymap.of([...defaultKeymap, ...historyKeymap]),
        updateListener,
      ],
    })

    const view = new EditorView({ state, parent: containerRef.current })
    viewRef.current = view

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current)
      view.destroy()
      viewRef.current = null
    }
  }, [updateScene, setSaveState])

  return { containerRef, loadScene, saveCursorState }
}