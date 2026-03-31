import { useState } from 'react'
import { useInsightsStore } from '../state/insightsStore'
import { useSceneStore } from '../state/sceneStore'
import { AnalyzeScene, GetSettings, SaveSettings } from '../../bindings/story-engine/app'
import type { AppSettings } from '../types'

// Opacity class for character name by rank (0 = most frequent).
function charOpacity(rank: number): string {
  const classes = ['text-stone-bright', 'text-stone-text', 'text-stone-subtle', 'text-stone-muted']
  return classes[Math.min(rank, classes.length - 1)]
}

// One-line scene characterisation from tone, shown only for LLM results.
function sceneLine(tone: string): string {
  const lines: Record<string, string> = {
    tense:   'Something tense runs through this.',
    warm:    'A warm scene.',
    urgent:  'Urgency here.',
    quiet:   'Quiet.',
    neutral: 'A measured exchange.',
  }
  return lines[tone] ?? ''
}

export function InsightsPanel() {
  const { mirror, isAnalyzing, analysisError, setAnalyzing, setAnalysisError } =
    useInsightsStore()
  const { scenes, activeSceneId } = useSceneStore()

  const [showSettings, setShowSettings] = useState(false)
  const [settings, setSettings] = useState<AppSettings>({
    llmEndpoint: 'http://localhost:11434/v1',
    llmModel: 'llama3.2:3b',
    llmApiKey: '',
  })
  const [settingsLoaded, setSettingsLoaded] = useState(false)
  const [savingSettings, setSavingSettings] = useState(false)

  const activeScene = scenes.find((s) => s.id === activeSceneId) ?? null
  const wordCount = activeScene?.wordCount ?? 0

  const hasLLMData = mirror?.source === 'llm'
  const hasEntities = (mirror?.entities?.length ?? 0) > 0
  const hasInteractions = hasLLMData && (mirror?.interactions?.length ?? 0) > 0

  // Load settings when the form is opened.
  const handleOpenSettings = async () => {
    if (!settingsLoaded) {
      try {
        const s = await GetSettings()
        setSettings(s)
        setSettingsLoaded(true)
      } catch (err) {
        console.error('[Story Engine] GetSettings failed:', err)
      }
    }
    setShowSettings((v) => !v)
  }

  const handleSaveSettings = async () => {
    setSavingSettings(true)
    try {
      await SaveSettings(settings)
      setShowSettings(false)
    } catch (err) {
      console.error('[Story Engine] SaveSettings failed:', err)
    } finally {
      setSavingSettings(false)
    }
  }

  const handleAnalyze = async () => {
    if (!activeSceneId || isAnalyzing) return
    setAnalyzing(true)
    setAnalysisError(null)
    try {
      await AnalyzeScene(activeSceneId)
      // Mirror updates automatically via the "mirror:updated" event in App.tsx.
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      setAnalysisError(msg)
    } finally {
      setAnalyzing(false)
    }
  }

  return (
    <div className="flex flex-col h-full bg-stone-surface border-l border-stone-border select-none w-64 shrink-0">

      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-stone-border">
        <span className="text-[10px] tracking-widest uppercase text-stone-muted font-ui">
          Mirror
        </span>
        <div className="flex items-center gap-2">
          {/* Brain icon — triggers LLM analysis */}
          <button
            onClick={handleAnalyze}
            disabled={!activeSceneId || isAnalyzing}
            title={isAnalyzing ? 'Analysing…' : 'Run narrative analysis'}
            className={`
              text-base leading-none transition-all
              ${isAnalyzing
                ? 'opacity-100 animate-pulse cursor-not-allowed'
                : 'opacity-40 hover:opacity-90 cursor-pointer'}
              ${analysisError ? 'text-red-400 opacity-100' : 'text-stone-text'}
              disabled:cursor-not-allowed
            `}
          >
            🧠
          </button>
          {/* Settings gear */}
          <button
            onClick={handleOpenSettings}
            title="LLM settings"
            className="text-xs text-stone-muted hover:text-stone-text transition-colors"
          >
            ⚙
          </button>
        </div>
      </div>

      {/* Main body */}
      <div className="flex-1 overflow-y-auto px-4 py-4 flex flex-col gap-4">

        {/* Empty state — no analysis yet */}
        {mirror === null && (
          <p className="text-stone-muted text-xs font-ui italic">
            Click 🧠 to run analysis.
          </p>
        )}

        {/* Error state */}
        {analysisError && (
          <div className="rounded px-2 py-2 bg-stone-border">
            <p className="text-red-400 text-[10px] font-ui leading-relaxed">
              {analysisError.includes('Ollama') || analysisError.includes('connection')
                ? 'Could not reach the LLM. Is Ollama running?'
                : analysisError}
            </p>
          </div>
        )}

        {/* Scene character line — LLM only */}
        {hasLLMData && mirror?.sceneTone && (
          <p className="text-stone-subtle text-xs font-ui italic leading-relaxed">
            {sceneLine(mirror.sceneTone)}
          </p>
        )}

        {/* Characters present */}
        {hasEntities && mirror && (
          <div>
            <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui mb-2">
              Present
            </p>
            <div className="flex flex-col gap-1">
              {mirror.entities.map((name, i) => (
                <span key={name} className={`text-sm font-ui ${charOpacity(i)}`}>
                  {name}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Rule-based hint — nudge toward LLM */}
        {mirror?.source === 'rule' && hasEntities && (
          <p className="text-stone-muted text-[10px] font-ui italic">
            Click 🧠 for interaction summaries.
          </p>
        )}

        {/* Divider */}
        {hasEntities && hasInteractions && (
          <div className="border-t border-stone-border" />
        )}

        {/* Interaction summaries — LLM only */}
        {hasInteractions && mirror && (
          <div>
            <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui mb-3">
              Exchanges
            </p>
            <div className="flex flex-col gap-3">
              {mirror.interactions.map((interaction, i) => (
                <p key={i} className="text-xs font-ui text-stone-subtle italic leading-relaxed">
                  {interaction.summary}
                </p>
              ))}
            </div>
          </div>
        )}

        {/* No interactions from LLM */}
        {hasLLMData && !hasInteractions && hasEntities && (
          <p className="text-stone-muted text-xs font-ui italic">
            No exchanges detected.
          </p>
        )}

      </div>

      {/* Settings form */}
      {showSettings && (
        <div className="border-t border-stone-border px-4 py-4 flex flex-col gap-3">
          <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui">
            LLM Settings
          </p>

          <div className="flex flex-col gap-1">
            <label className="text-[10px] text-stone-muted font-ui">Endpoint</label>
            <input
              type="text"
              value={settings.llmEndpoint}
              onChange={(e) => setSettings({ ...settings, llmEndpoint: e.target.value })}
              placeholder="http://localhost:11434/v1"
              className="bg-stone-border text-stone-text text-xs font-ui px-2 py-1.5 rounded outline-none focus:ring-1 focus:ring-stone-muted w-full"
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-[10px] text-stone-muted font-ui">Model</label>
            <input
              type="text"
              value={settings.llmModel}
              onChange={(e) => setSettings({ ...settings, llmModel: e.target.value })}
              placeholder="llama3.2:3b"
              className="bg-stone-border text-stone-text text-xs font-ui px-2 py-1.5 rounded outline-none focus:ring-1 focus:ring-stone-muted w-full"
            />
          </div>

          <div className="flex flex-col gap-1">
            <label className="text-[10px] text-stone-muted font-ui">
              API Key <span className="text-stone-muted">(optional)</span>
            </label>
            <input
              type="password"
              value={settings.llmApiKey}
              onChange={(e) => setSettings({ ...settings, llmApiKey: e.target.value })}
              placeholder="Leave empty for local models"
              className="bg-stone-border text-stone-text text-xs font-ui px-2 py-1.5 rounded outline-none focus:ring-1 focus:ring-stone-muted w-full"
            />
          </div>

          <button
            onClick={handleSaveSettings}
            disabled={savingSettings}
            className="mt-1 bg-stone-border hover:bg-stone-muted text-stone-text text-xs font-ui py-1.5 rounded transition-colors disabled:opacity-50"
          >
            {savingSettings ? 'Saving…' : 'Save'}
          </button>
        </div>
      )}

      {/* Word count */}
      {wordCount > 0 && !showSettings && (
        <div className="px-4 py-2 border-t border-stone-border">
          <span className="text-[10px] text-stone-muted font-ui">
            {wordCount.toLocaleString()} words
          </span>
        </div>
      )}

    </div>
  )
}