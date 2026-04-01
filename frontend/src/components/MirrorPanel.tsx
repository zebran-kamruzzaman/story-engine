import { useState, useRef, useEffect, useCallback } from 'react'
import { useMirrorStore } from '../state/mirrorStore'
import { useSceneStore } from '../state/sceneStore'
import {
  AnalyzeProject,
  AnalyzeScene,
  AskQuestion,
  GetChatHistory,
  ClearChat,
} from '../../bindings/story-engine/app'
import type { ChatMessage } from '../types'

// ─── Section wrapper ─────────────────────────────────────────────────────────

function Section({
  label,
  defaultOpen = true,
  action,
  children,
}: {
  label: string
  defaultOpen?: boolean
  action?: React.ReactNode
  children: React.ReactNode
}) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="border-b border-stone-border">
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center justify-between px-4 py-2.5 text-left hover:bg-stone-border/30 transition-colors"
      >
        <span className="text-[10px] uppercase tracking-widest text-stone-muted font-ui">
          {open ? '▾' : '▸'} {label}
        </span>
        {action && <span onClick={(e) => e.stopPropagation()}>{action}</span>}
      </button>
      {open && <div className="px-4 pb-3">{children}</div>}
    </div>
  )
}

// ─── Character row ───────────────────────────────────────────────────────────

function CharacterRow({ name, description }: { name: string; description: string }) {
  const [expanded, setExpanded] = useState(false)
  return (
    <div className="mb-1">
      <button
        onClick={() => setExpanded((v) => !v)}
        className="w-full flex items-center gap-1.5 py-1 text-left group"
      >
        <span className="text-[10px] text-stone-muted font-ui">
          {expanded ? '▾' : '▸'}
        </span>
        <span className="text-xs font-ui text-stone-text group-hover:text-stone-bright transition-colors">
          {name}
        </span>
      </button>
      {expanded && (
        <div className="ml-4 mt-1">
          {description ? (
            <p className="text-[11px] font-ui text-stone-subtle leading-relaxed italic">
              {description}
            </p>
          ) : (
            <p className="text-[11px] font-ui text-stone-muted italic">
              No description yet. Click 🧠 above to analyze.
            </p>
          )}
        </div>
      )}
    </div>
  )
}

// ─── Chat message ────────────────────────────────────────────────────────────

function ChatBubble({ msg }: { msg: ChatMessage }) {
  const isUser = msg.role === 'user'
  return (
    <div className={`mb-3 ${isUser ? 'text-right' : 'text-left'}`}>
      <div
        className={`inline-block text-xs font-ui leading-relaxed rounded px-2.5 py-2 max-w-[90%] text-left ${
          isUser
            ? 'bg-stone-border text-stone-text'
            : 'bg-transparent text-stone-subtle'
        }`}
      >
        {msg.content}
      </div>
      {msg.sources && msg.sources.length > 0 && (
        <div className="mt-1 text-[10px] text-stone-muted font-ui">
          {msg.sources.map((s, i) => (
            <span key={i} className="mr-1">
              {i === 0 ? 'from: ' : ''}
              <span className="italic">{s.title}</span>
              {i < msg.sources!.length - 1 ? ',' : ''}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── MirrorPanel ─────────────────────────────────────────────────────────────

export function MirrorPanel() {
  const { scenes, activeSceneId } = useSceneStore()
  const {
    characters,
    sceneSummaries,
    chatHistory,
    isAnalyzingProject,
    isAnalyzingScene,
    analysisError,
    setChatHistory,
    appendChatMessage,
    setAnalyzingProject,
    setAnalyzingScene,
    setAnalysisError,
    clearError,
  } = useMirrorStore()

  const activeScene = scenes.find((s) => s.id === activeSceneId) ?? null
  const currentSummary = activeSceneId ? (sceneSummaries[activeSceneId] ?? '') : ''

  const [question, setQuestion] = useState('')
  const [isAsking, setIsAsking] = useState(false)
  const chatEndRef = useRef<HTMLDivElement>(null)

  // Load chat history from disk once on mount.
  const loadChatHistory = useCallback(async () => {
    try {
      const history = await GetChatHistory()
      setChatHistory(history ?? [])
    } catch (err) {
      console.error('[Story Engine] GetChatHistory failed:', err)
    }
  }, [setChatHistory])

  useEffect(() => {
    loadChatHistory()
  }, [loadChatHistory])

  // Auto-scroll chat to bottom when new messages arrive.
  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [chatHistory])

  // ── Project analysis ──────────────────────────────────────────────────────

  const handleAnalyzeProject = async () => {
    setAnalyzingProject(true)
    clearError()
    try {
      await AnalyzeProject()
    } catch (err: unknown) {
      setAnalysisError(err instanceof Error ? err.message : String(err))
    } finally {
      setAnalyzingProject(false)
    }
  }

  // ── Scene analysis ────────────────────────────────────────────────────────

  const handleAnalyzeScene = async () => {
    if (!activeSceneId) return
    setAnalyzingScene(true)
    clearError()
    try {
      await AnalyzeScene(activeSceneId)
    } catch (err: unknown) {
      setAnalysisError(err instanceof Error ? err.message : String(err))
    } finally {
      setAnalyzingScene(false)
    }
  }

  // ── Chat ─────────────────────────────────────────────────────────────────

  const handleAsk = async () => {
    const q = question.trim()
    if (!q || isAsking) return
    setQuestion('')
    setIsAsking(true)
    appendChatMessage({ role: 'user', content: q })
    try {
      const response = await AskQuestion(q)
      appendChatMessage({
        role: 'assistant',
        content: response.answer,
        sources: response.sources,
      })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Something went wrong.'
      appendChatMessage({ role: 'assistant', content: msg })
    } finally {
      setIsAsking(false)
    }
  }

  const handleClearChat = async () => {
    try {
      await ClearChat()
      setChatHistory([])
    } catch (err) {
      console.error('[Story Engine] ClearChat failed:', err)
    }
  }

  const characterNames = Object.keys(characters).sort()

  return (
    <div className="flex flex-col h-full bg-stone-surface border-l border-stone-border select-none w-72 shrink-0">
      {/* Panel header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-stone-border">
        <span className="text-[10px] tracking-widest uppercase text-stone-muted font-ui">
          Mirror
        </span>
        {analysisError && (
          <span
            className="text-[10px] text-red-400 font-ui cursor-pointer"
            onClick={clearError}
            title={analysisError}
          >
            ⚠ Error
          </span>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        {/* ── Zone 1: Characters (project-wide) ─────────────────────────── */}
        <Section
          label="Characters"
          defaultOpen={true}
          action={
            <button
              onClick={handleAnalyzeProject}
              disabled={isAnalyzingProject}
              title="Analyze all scenes to build character roster"
              className={`text-base transition-opacity ${
                isAnalyzingProject
                  ? 'animate-pulse opacity-100'
                  : 'opacity-40 hover:opacity-90'
              }`}
            >
              🧠
            </button>
          }
        >
          {characterNames.length === 0 ? (
            <p className="text-stone-muted text-[11px] font-ui italic">
              {isAnalyzingProject
                ? 'Analysing…'
                : 'Click 🧠 to build the character roster.'}
            </p>
          ) : (
            characterNames.map((name) => (
              <CharacterRow
                key={name}
                name={name}
                description={characters[name]?.description ?? ''}
              />
            ))
          )}
        </Section>

        {/* ── Zone 2: Scene Summary ────────────────────────────────────────── */}
        <Section
          label={activeScene ? `This scene: ${activeScene.title}` : 'Scene Summary'}
          defaultOpen={true}
          action={
            <button
              onClick={handleAnalyzeScene}
              disabled={isAnalyzingScene || !activeSceneId}
              title="Summarize this scene"
              className={`text-base transition-opacity ${
                isAnalyzingScene
                  ? 'animate-pulse opacity-100'
                  : 'opacity-40 hover:opacity-90'
              }`}
            >
              🧠
            </button>
          }
        >
          {!activeSceneId ? (
            <p className="text-stone-muted text-[11px] font-ui italic">
              Select a scene to see its summary.
            </p>
          ) : currentSummary ? (
            <p className="text-[11px] font-ui text-stone-subtle leading-relaxed">
              {isAnalyzingScene ? (
                <span className="italic text-stone-muted">Updating…</span>
              ) : (
                currentSummary
              )}
            </p>
          ) : (
            <p className="text-stone-muted text-[11px] font-ui italic">
              {isAnalyzingScene
                ? 'Summarizing…'
                : 'No summary yet. Click 🧠 or write ~100 words to trigger auto-summary.'}
            </p>
          )}
        </Section>

        {/* ── Zone 3: Chat ─────────────────────────────────────────────────── */}
        <Section
          label="Ask the Story"
          defaultOpen={true}
          action={
            chatHistory.length > 0 ? (
              <button
                onClick={handleClearChat}
                className="text-[10px] text-stone-muted hover:text-stone-text font-ui transition-colors"
                title="Clear chat"
              >
                Clear
              </button>
            ) : undefined
          }
        >
          <div className="mb-3 max-h-72 overflow-y-auto">
            {chatHistory.length === 0 ? (
              <p className="text-stone-muted text-[11px] font-ui italic">
                Ask anything about your story.
              </p>
            ) : (
              <>
                {chatHistory.map((msg, i) => (
                  <ChatBubble key={i} msg={msg} />
                ))}
                {isAsking && (
                  <p className="text-[11px] text-stone-muted font-ui italic animate-pulse">
                    Thinking…
                  </p>
                )}
                <div ref={chatEndRef} />
              </>
            )}
          </div>

          <div className="flex flex-col gap-1.5">
            <textarea
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleAsk()
                }
              }}
              placeholder="Ask about your story… (Enter to send)"
              rows={2}
              disabled={isAsking}
              className="bg-stone-border text-stone-text text-xs font-ui px-2 py-1.5 rounded outline-none focus:ring-1 focus:ring-stone-muted resize-none disabled:opacity-50"
              style={{ userSelect: 'text' }}
            />
            <button
              onClick={handleAsk}
              disabled={isAsking || !question.trim()}
              className="bg-stone-border hover:bg-stone-muted text-stone-text text-xs font-ui py-1 rounded transition-colors disabled:opacity-50"
            >
              {isAsking ? 'Asking…' : 'Ask'}
            </button>
          </div>
        </Section>
      </div>

      {/* Word count footer */}
      {activeScene && activeScene.wordCount > 0 && (
        <div className="px-4 py-2 border-t border-stone-border shrink-0">
          <span className="text-[10px] text-stone-muted font-ui">
            {activeScene.wordCount.toLocaleString()} words in this scene
          </span>
        </div>
      )}
    </div>
  )
}