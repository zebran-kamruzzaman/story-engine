import { useInsightsStore } from '../state/insightsStore'
import { useSceneStore } from '../state/sceneStore'

export function InsightsPanel() {
  const insights = useInsightsStore((s) => s.insights)
  const { scenes, activeSceneId } = useSceneStore()

  const activeScene = scenes.find((s) => s.id === activeSceneId) ?? null
  const currentWordCount = activeScene?.wordCount ?? 0

  return (
    <div className="flex flex-col h-full bg-stone-surface border-l border-stone-border select-none w-64 shrink-0">
      {/* Header */}
      <div className="px-3 py-3 border-b border-stone-border">
        <span className="text-xs tracking-widest uppercase text-stone-muted font-ui">
          Insights
        </span>
      </div>

      <div className="flex-1 overflow-y-auto px-3 py-3 space-y-5">
        {insights === null ? (
          <p className="text-stone-subtle text-xs font-ui italic">
            Waiting for analysis...
          </p>
        ) : (
          <>
            {/* Characters */}
            <div>
              <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui mb-2">
                Characters
              </p>
              {insights.entities.length === 0 ? (
                <p className="text-stone-subtle text-xs font-ui italic">
                  None detected
                </p>
              ) : (
                <div className="flex flex-wrap gap-1.5">
                  {insights.entities.map((name) => (
                    <span
                      key={name}
                      className="bg-stone-border text-stone-text text-xs rounded px-2 py-0.5 font-ui"
                    >
                      {name}
                    </span>
                  ))}
                </div>
              )}
            </div>

            {/* Dialogue */}
            <div>
              <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui mb-1">
                Dialogue
              </p>
              <p className="text-stone-text text-xs font-ui">
                <span className="text-stone-bright font-medium">
                  {insights.dialogueCount}
                </span>{' '}
                dialogue segment{insights.dialogueCount !== 1 ? 's' : ''} detected
              </p>
            </div>

            {/* Scene stats */}
            <div>
              <p className="text-[10px] uppercase tracking-widest text-stone-muted font-ui mb-1">
                Scene Stats
              </p>
              <p className="text-stone-text text-xs font-ui">
                <span className="text-stone-bright font-medium">
                  {currentWordCount.toLocaleString()}
                </span>{' '}
                words
              </p>
            </div>
          </>
        )}
      </div>
    </div>
  )
}