package services

import (
	"encoding/json"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventService dispatches events to the React frontend via ExecJS.
// App.tsx listens with window.addEventListener('mirror:updated', handler).
type EventService struct {
	window *application.WebviewWindow
}

func NewEventService(window *application.WebviewWindow) *EventService {
	return &EventService{window: window}
}

// mirrorEvent is the JSON shape sent to the frontend.
// interactionEntry mirrors models.CharacterInteraction without importing models.
type mirrorEvent struct {
	SceneID      string             `json:"sceneId"`
	Entities     []string           `json:"entities"`
	Interactions []interactionEntry `json:"interactions"`
	SceneTone    string             `json:"sceneTone"`
	Source       string             `json:"source"`
}

type interactionEntry struct {
	Characters []string `json:"characters"`
	Tone       string   `json:"tone"`
	Summary    string   `json:"summary"`
}

// EmitMirrorUpdated fires the "mirror:updated" CustomEvent in the WebView.
// interactions may be nil for rule-based updates (only entities are populated).
// source is "rule" for background analysis and "llm" for LLM analysis.
func (s *EventService) EmitMirrorUpdated(
	sceneID string,
	entities []string,
	interactions interface{},
	sceneTone string,
	source string,
) {
	if s.window == nil {
		return
	}
	if entities == nil {
		entities = []string{}
	}

	// Normalise interactions to our event type.
	var entries []interactionEntry
	if interactions != nil {
		// interactions comes from LLMResult.Interactions — marshal/unmarshal
		// to convert the anonymous struct slice to interactionEntry slice.
		b, err := json.Marshal(interactions)
		if err == nil {
			_ = json.Unmarshal(b, &entries)
		}
	}
	if entries == nil {
		entries = []interactionEntry{}
	}

	payload := mirrorEvent{
		SceneID:      sceneID,
		Entities:     entities,
		Interactions: entries,
		SceneTone:    sceneTone,
		Source:       source,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	js := fmt.Sprintf(
		`window.dispatchEvent(new CustomEvent('mirror:updated', {detail: %s}))`,
		string(data),
	)
	s.window.ExecJS(js)
}
