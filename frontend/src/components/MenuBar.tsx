import { useState } from 'react'
import {
  CreateProject,
  SwitchProject,
  GetSettings,
  SaveSettings,
} from '../../bindings/story-engine/app'
import { useProjectStore } from '../state/projectStore'
import { useSceneStore } from '../state/sceneStore'
import { useMirrorStore } from '../state/mirrorStore'
import type { AppSettings } from '../types'

export function MenuBar() {
  const { currentProject, projects, setCurrentProject, setProjects } =
    useProjectStore()
  const { setScenes, setActiveSceneId } = useSceneStore()
  const { setCharacters, setChatHistory } = useMirrorStore()

  const [showProjectList, setShowProjectList] = useState(false)
  const [showNewProject, setShowNewProject] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [newProjectName, setNewProjectName] = useState('')
  const [settings, setSettings] = useState<AppSettings | null>(null)
  const [saving, setSaving] = useState(false)
  const [creating, setCreating] = useState(false)
  const [switching, setSwitching] = useState(false)

  const resetProjectState = () => {
    setCharacters({})
    setChatHistory([])
    setActiveSceneId(null)
  }

  const handleSwitchProject = async (path: string) => {
    setSwitching(true)
    setShowProjectList(false)
    resetProjectState()
    try {
      const scenes = await SwitchProject(path)
      const list = scenes ?? []
      setScenes(list)
      const proj = projects.find((p) => p.path === path)
      if (proj) setCurrentProject(proj)
      if (list.length > 0) setActiveSceneId(list[0].id)
    } catch (err) {
      console.error('[Story Engine] SwitchProject failed:', err)
    } finally {
      setSwitching(false)
    }
  }

  const handleCreateProject = async () => {
    const name = newProjectName.trim()
    if (!name) return
    setCreating(true)
    try {
      const info = await CreateProject(name)
      setScenes([])
      setCurrentProject(info)
      setProjects([...projects, info])
      setActiveSceneId(null)
      setShowNewProject(false)
      setNewProjectName('')
      resetProjectState()
    } catch (err) {
      console.error('[Story Engine] CreateProject failed:', err)
    } finally {
      setCreating(false)
    }
  }

  const handleOpenSettings = async () => {
    if (!settings) {
      try {
        const s = await GetSettings()
        // GetSettings returns the binding type which includes lastProjectPath.
        // Map it to our UI-only AppSettings type (which omits lastProjectPath).
        setSettings({
          llmEndpoint: s.llmEndpoint,
          llmModel: s.llmModel,
          llmApiKey: s.llmApiKey,
        })
      } catch (err) {
        console.error('[Story Engine] GetSettings failed:', err)
      }
    }
    setShowSettings((v) => !v)
    setShowProjectList(false)
    setShowNewProject(false)
  }

  const handleSaveSettings = async () => {
    if (!settings) return
    setSaving(true)
    try {
      // Pass lastProjectPath: '' — Go's SaveSettings always preserves the
      // real value server-side (settings.LastProjectPath = a.settings.LastProjectPath).
      await SaveSettings({ ...settings, lastProjectPath: '' })
      setShowSettings(false)
    } catch (err) {
      console.error('[Story Engine] SaveSettings failed:', err)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="relative shrink-0">
      {/* Main bar */}
      <div className="flex items-center justify-between px-4 h-9 bg-stone-surface border-b border-stone-border select-none">
        <span className="text-[10px] uppercase tracking-widest text-stone-muted font-ui">
          Story Engine
        </span>
        <span className="text-xs font-ui text-stone-text">
          {switching ? 'Opening…' : (currentProject?.name ?? '—')}
        </span>
        <div className="flex items-center gap-3">
          <button
            onClick={() => {
              setShowNewProject((v) => !v)
              setShowProjectList(false)
              setShowSettings(false)
            }}
            className="text-[10px] font-ui text-stone-subtle hover:text-stone-text transition-colors"
          >
            + New Story
          </button>
          <button
            onClick={() => {
              setShowProjectList((v) => !v)
              setShowNewProject(false)
              setShowSettings(false)
            }}
            className="text-[10px] font-ui text-stone-subtle hover:text-stone-text transition-colors"
          >
            Open Story
          </button>
          <button
            onClick={handleOpenSettings}
            className="text-[10px] font-ui text-stone-muted hover:text-stone-text transition-colors"
            title="Settings"
          >
            ⚙
          </button>
        </div>
      </div>

      {/* New project form */}
      {showNewProject && (
        <div className="absolute right-0 top-9 z-50 bg-stone-surface border border-stone-border rounded-b shadow-lg p-4 w-64 flex flex-col gap-2">
          <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui">
            New Story
          </p>
          <input
            type="text"
            placeholder="Story title…"
            value={newProjectName}
            onChange={(e) => setNewProjectName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleCreateProject()
              if (e.key === 'Escape') setShowNewProject(false)
            }}
            autoFocus
            className="bg-stone-border text-stone-text text-xs font-ui px-2 py-1.5 rounded outline-none focus:ring-1 focus:ring-stone-muted"
            style={{ userSelect: 'text' }}
          />
          <button
            onClick={handleCreateProject}
            disabled={creating || !newProjectName.trim()}
            className="bg-stone-border hover:bg-stone-muted text-stone-text text-xs font-ui py-1.5 rounded transition-colors disabled:opacity-50"
          >
            {creating ? 'Creating…' : 'Create'}
          </button>
        </div>
      )}

      {/* Project list dropdown */}
      {showProjectList && (
        <div className="absolute right-0 top-9 z-50 bg-stone-surface border border-stone-border rounded-b shadow-lg w-64 max-h-64 overflow-y-auto">
          <p className="px-3 py-2 text-[10px] uppercase tracking-widest text-stone-muted font-ui border-b border-stone-border">
            Your Stories
          </p>
          {projects.length === 0 ? (
            <p className="px-3 py-3 text-xs font-ui text-stone-subtle italic">
              No stories found.
            </p>
          ) : (
            projects.map((p) => (
              <button
                key={p.path}
                onClick={() => handleSwitchProject(p.path)}
                className={`w-full text-left px-3 py-2 text-xs font-ui transition-colors hover:bg-stone-border ${
                  p.path === currentProject?.path
                    ? 'text-stone-bright'
                    : 'text-stone-text'
                }`}
              >
                {p.name}
                {p.path === currentProject?.path && (
                  <span className="ml-2 text-[10px] text-stone-muted">current</span>
                )}
              </button>
            ))
          )}
        </div>
      )}

      {/* Settings form */}
      {showSettings && settings && (
        <div className="absolute right-0 top-9 z-50 bg-stone-surface border border-stone-border rounded-b shadow-lg p-4 w-72 flex flex-col gap-3">
          <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui">
            LLM Settings
          </p>
          {(
            [
              { label: 'Endpoint', key: 'llmEndpoint' as const, placeholder: 'http://localhost:11434/v1', type: 'text' },
              { label: 'Model', key: 'llmModel' as const, placeholder: 'llama3.2:3b', type: 'text' },
              { label: 'API Key (optional)', key: 'llmApiKey' as const, placeholder: 'Leave empty for local', type: 'password' },
            ] as const
          ).map(({ label, key, placeholder, type }) => (
            <div key={key} className="flex flex-col gap-1">
              <label className="text-[10px] text-stone-muted font-ui">{label}</label>
              <input
                type={type}
                value={settings[key]}
                placeholder={placeholder}
                onChange={(e) => setSettings({ ...settings, [key]: e.target.value })}
                className="bg-stone-border text-stone-text text-xs font-ui px-2 py-1.5 rounded outline-none focus:ring-1 focus:ring-stone-muted"
                style={{ userSelect: 'text' }}
              />
            </div>
          ))}
          <button
            onClick={handleSaveSettings}
            disabled={saving}
            className="bg-stone-border hover:bg-stone-muted text-stone-text text-xs font-ui py-1.5 rounded transition-colors disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      )}
    </div>
  )
}