package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"story-engine/internal/models"
)

// LLMService calls an OpenAI-compatible chat completions endpoint.
// Any provider that implements POST /v1/chat/completions works:
//   - Ollama:    http://localhost:11434/v1  (no API key)
//   - LM Studio: http://localhost:1234/v1   (no API key)
//   - OpenAI:    https://api.openai.com/v1  (API key required)
//   - Groq:      https://api.groq.com/openai/v1 (API key required)
type LLMService struct {
	mu     sync.RWMutex
	config models.AppSettings
	client *http.Client
}

// NewLLMService creates an LLMService with the given settings.
func NewLLMService(settings models.AppSettings) *LLMService {
	return &LLMService{
		config: settings,
		// 90-second timeout: generous enough for a slow 7B model on CPU,
		// strict enough to not hang forever if Ollama is not running.
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// UpdateConfig hot-swaps the LLM configuration without restarting the service.
// Called when the writer saves new settings in the panel.
func (s *LLMService) UpdateConfig(settings models.AppSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = settings
}

// ── Request / response types for the OpenAI chat completions API ─────────────

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ── LLM analysis result (internal; not exported via IPC) ─────────────────────

// LLMResult is the parsed output from the LLM analysis prompt.
type LLMResult struct {
	Characters   []string `json:"characters"`
	SceneTone    string   `json:"sceneTone"`
	Interactions []struct {
		Characters []string `json:"characters"`
		Tone       string   `json:"tone"`
		Summary    string   `json:"summary"`
	} `json:"interactions"`
}

// ── Public API ────────────────────────────────────────────────────────────────

// AnalyzeScene sends the scene text to the LLM and returns a structured analysis.
// It returns an error if the LLM endpoint is unreachable, the model is not loaded,
// or the response cannot be parsed as valid JSON.
func (s *LLMService) AnalyzeScene(content string) (*LLMResult, error) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if cfg.LLMEndpoint == "" || cfg.LLMModel == "" {
		return nil, fmt.Errorf("LLM not configured: set endpoint and model in settings")
	}

	prompt := buildPrompt(content)

	reqBody := chatRequest{
		Model: cfg.LLMModel,
		Messages: []chatMessage{
			{Role: "system", Content: "You are a literary analysis assistant. Return only valid JSON."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.2, // Low temperature for consistent structured output.
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	url := strings.TrimRight(cfg.LLMEndpoint, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.LLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.LLMAPIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: request failed (is Ollama running?): %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("llm: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBytes, &chatResp); err != nil {
		return nil, fmt.Errorf("llm: parse chat response: %w", err)
	}
	if chatResp.Error != nil {
		return nil, fmt.Errorf("llm: provider error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("llm: empty response from model")
	}

	raw := chatResp.Choices[0].Message.Content
	result, err := parseAnalysisJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("llm: parse analysis JSON: %w", err)
	}

	return result, nil
}

// ── Prompt construction ────────────────────────────────────────────────────────

// buildPrompt constructs the analysis prompt. The prompt is designed to work
// reliably with small models (3B parameters) by being explicit, concrete, and
// requesting the exact JSON schema in the prompt itself.
//
// Temperature 0.2 and the strict JSON-only instruction help small models stay
// on task without generating prose before or after the JSON object.
func buildPrompt(content string) string {
	// Truncate very long scenes to avoid context window limits on small models.
	// 3000 characters covers roughly 600 words — most scene drafts. The analysis
	// is based on character presence and dialogue patterns which are detectable
	// from the first portion of a scene if it's very long.
	if len(content) > 3000 {
		content = content[:3000] + "\n[...scene continues]"
	}

	return `Analyze the following fiction scene. Return ONLY a valid JSON object with no markdown, no explanation, no text before or after.

Required JSON format:
{
  "characters": ["Name1", "Name2"],
  "sceneTone": "tense",
  "interactions": [
    {
      "characters": ["Name1", "Name2"],
      "tone": "tense",
      "summary": "Elena and Marcus — a heated confrontation."
    }
  ]
}

Rules:
- characters: array of named characters who appear (first names preferred, max 8)
- sceneTone: ONE of: tense, warm, urgent, quiet, neutral
- interactions: one entry per distinct pair or group with meaningful exchange or shared focus
  - If only one character speaks or acts significantly alone, use one-element array: ["Name"]
  - summary: 6 to 12 words written as a margin note. Examples:
    "Elena and Marcus — a heated argument over the map."
    "Lyra speaks at length; Doran listens in silence."
    "A tense standoff between Elena, Marcus, and the stranger."
- If no interactions are present, return an empty array: "interactions": []
- sceneTone must be a single word from the allowed list

Scene:
` + content
}

// ── JSON parsing ───────────────────────────────────────────────────────────────

// parseAnalysisJSON strips any markdown code fences and parses the LLM output.
// Small models sometimes wrap JSON in ```json ... ``` even when instructed not to.
func parseAnalysisJSON(raw string) (*LLMResult, error) {
	raw = strings.TrimSpace(raw)

	// Strip markdown code fences.
	if strings.HasPrefix(raw, "```") {
		lines := strings.SplitN(raw, "\n", 2)
		if len(lines) > 1 {
			raw = lines[1]
		}
		if idx := strings.LastIndex(raw, "```"); idx >= 0 {
			raw = strings.TrimSpace(raw[:idx])
		}
	}

	// Find the outermost JSON object in case there's prose before/after.
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}

	var result LLMResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// Include the first 120 chars of raw output in the error to aid debugging.
		preview := raw
		if len(preview) > 120 {
			preview = preview[:120] + "..."
		}
		return nil, fmt.Errorf("JSON unmarshal failed: %w — raw: %q", err, preview)
	}

	// Sanitize: ensure slice fields are non-nil for JSON serialization.
	if result.Characters == nil {
		result.Characters = []string{}
	}
	if result.Interactions == nil {
		result.Interactions = []struct {
			Characters []string `json:"characters"`
			Tone       string   `json:"tone"`
			Summary    string   `json:"summary"`
		}{}
	}

	// Validate sceneTone against allowed values; default to "neutral".
	allowed := map[string]bool{"tense": true, "warm": true, "urgent": true, "quiet": true, "neutral": true}
	if !allowed[result.SceneTone] {
		result.SceneTone = "neutral"
	}

	return &result, nil
}
