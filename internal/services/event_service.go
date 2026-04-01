package services

import (
	"encoding/json"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventService dispatches events to the React frontend via ExecJS CustomEvents.
type EventService struct {
	window *application.WebviewWindow
}

func NewEventService(window *application.WebviewWindow) *EventService {
	return &EventService{window: window}
}

func (s *EventService) emit(name string, payload interface{}) {
	if s.window == nil {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	s.window.ExecJS(fmt.Sprintf(
		`window.dispatchEvent(new CustomEvent(%q, {detail: %s}))`,
		name, string(data),
	))
}

// EmitEntitiesUpdated fires after background entity detection completes.
// The frontend uses this to keep the character roster name list up to date
// without requiring an LLM call.
func (s *EventService) EmitEntitiesUpdated(sceneID string, entities []string) {
	if entities == nil {
		entities = []string{}
	}
	s.emit("mirror:entities-updated", map[string]interface{}{
		"sceneId":  sceneID,
		"entities": entities,
	})
}

// EmitCharactersUpdated fires after AnalyzeProject() completes.
// Carries the full updated character roster.
func (s *EventService) EmitCharactersUpdated(roster map[string]interface{}) {
	s.emit("mirror:characters-updated", roster)
}

// EmitSceneSummaryUpdated fires after a scene summary is generated.
func (s *EventService) EmitSceneSummaryUpdated(sceneID, summary string) {
	s.emit("mirror:scene-summary-updated", map[string]interface{}{
		"sceneId": sceneID,
		"summary": summary,
	})
}
