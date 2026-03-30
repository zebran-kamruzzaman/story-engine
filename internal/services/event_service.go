package services

import (
	"encoding/json"
	"fmt"

	"story-engine/internal/models"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// EventService dispatches events from the Go backend to the React frontend.
//
// Wails v3 alpha.74 has unstable event types (WailsEvent, CustomEvent, etc.
// keep changing between alphas). To avoid this entirely, we use ExecJS to
// fire a standard DOM CustomEvent that the frontend listens to with
// window.addEventListener — zero dependency on any alpha event API.
type EventService struct {
	window *application.WebviewWindow
}

func NewEventService(window *application.WebviewWindow) *EventService {
	return &EventService{window: window}
}

// EmitInsightsUpdated fires a DOM CustomEvent named "insights:updated" in the
// WebView. App.tsx listens for it with window.addEventListener.
func (s *EventService) EmitInsightsUpdated(sceneID string, entities []string, dialogueCount int) {
	if s.window == nil {
		return
	}

	payload := models.InsightsPayload{
		SceneID:       sceneID,
		Entities:      entities,
		DialogueCount: dialogueCount,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Dispatch a standard browser CustomEvent. The frontend catches this with
	// window.addEventListener('insights:updated', handler) in App.tsx.
	js := fmt.Sprintf(
		`window.dispatchEvent(new CustomEvent('insights:updated', {detail: %s}))`,
		string(data),
	)
	s.window.ExecJS(js)
}
