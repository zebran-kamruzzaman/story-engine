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
// Providers: Ollama (no key), LM Studio (no key), OpenAI, Groq, OpenRouter.
// Swap providers by changing endpoint + model in AppSettings — no code changes.
type LLMService struct {
	mu     sync.RWMutex
	config models.AppSettings
	client *http.Client
}

func NewLLMService(settings models.AppSettings) *LLMService {
	return &LLMService{
		config: settings,
		client: &http.Client{Timeout: 90 * time.Second},
	}
}

// UpdateConfig hot-swaps the LLM configuration without restarting.
func (s *LLMService) UpdateConfig(settings models.AppSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = settings
}

// ── OpenAI chat completions types ─────────────────────────────────────────────

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

// ── Core call ──────────────────────────────────────────────────────────────────

func (s *LLMService) call(systemPrompt, userPrompt string, temp float64) (string, error) {
	s.mu.RLock()
	cfg := s.config
	s.mu.RUnlock()

	if cfg.LLMEndpoint == "" || cfg.LLMModel == "" {
		return "", fmt.Errorf("LLM not configured: set endpoint and model in settings")
	}

	reqBody := chatRequest{
		Model: cfg.LLMModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: temp,
		Stream:      false,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("llm: marshal: %w", err)
	}

	url := strings.TrimRight(cfg.LLMEndpoint, "/") + "/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.LLMAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.LLMAPIKey)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: request failed (is Ollama running?): %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm: HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	var cr chatResponse
	if err := json.Unmarshal(respBytes, &cr); err != nil {
		return "", fmt.Errorf("llm: parse response: %w", err)
	}
	if cr.Error != nil {
		return "", fmt.Errorf("llm: provider error: %s", cr.Error.Message)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("llm: empty response from model")
	}
	return cr.Choices[0].Message.Content, nil
}

// ── Character roster analysis ──────────────────────────────────────────────────

// LLMCharacterResult is the parsed output from the character analysis prompt.
type LLMCharacterResult struct {
	Characters []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"characters"`
}

// AnalyzeCharacters sends all scene excerpts to the LLM and gets back a
// character roster with descriptions. sceneTexts maps scene title → content snippet.
func (s *LLMService) AnalyzeCharacters(projectName string, sceneTexts map[string]string) (*LLMCharacterResult, error) {
	var sb strings.Builder
	for title, content := range sceneTexts {
		snip := content
		if len(snip) > 1500 {
			snip = snip[:1500] + "\n[...continues]"
		}
		fmt.Fprintf(&sb, "\n[%s]\n%s\n", title, snip)
	}

	system := `You are a literary analyst. Return ONLY valid JSON, no markdown, no preamble.`
	user := fmt.Sprintf(`Identify all named characters in the story "%s". For each character write a 2-3 sentence description of who they are based solely on what the text shows.

Return ONLY this JSON:
{"characters":[{"name":"Name","description":"2-3 sentence description."}]}

Story excerpts:
%s`, projectName, sb.String())

	raw, err := s.call(system, user, 0.2)
	if err != nil {
		return nil, err
	}

	raw = extractJSON(raw)
	var result LLMCharacterResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("llm: parse character JSON: %w", err)
	}
	return &result, nil
}

// ── Scene summary ──────────────────────────────────────────────────────────────

// SummarizeScene generates a 2-4 sentence summary of a single scene.
func (s *LLMService) SummarizeScene(sceneTitle, content string) (string, error) {
	if len(content) > 3000 {
		content = content[:3000] + "\n[...scene continues]"
	}
	system := `You are a literary assistant. Write concise scene summaries.`
	user := fmt.Sprintf(`Summarize the following scene titled "%s" in 2-4 sentences. Focus on what happens, who is involved, and the emotional tone. Write as a neutral observer.

Scene:
%s`, sceneTitle, content)

	summary, err := s.call(system, user, 0.3)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(summary), nil
}

// ── Project chat ───────────────────────────────────────────────────────────────

// AskQuestion answers a question about the story using the provided scene context.
// sceneContext is a slice of (title, content) pairs, ranked by relevance.
// history is the last N chat turns for continuity.
func (s *LLMService) AskQuestion(
	projectName string,
	question string,
	sceneContext []struct{ Title, Content string },
	history []models.ChatMessage,
) (string, error) {
	var context strings.Builder
	for _, sc := range sceneContext {
		snip := sc.Content
		if len(snip) > 800 {
			snip = snip[:800] + "\n[...continues]"
		}
		fmt.Fprintf(&context, "\n[%s]\n%s\n", sc.Title, snip)
	}

	// Include recent history for conversational continuity (last 4 turns).
	var historyStr strings.Builder
	start := 0
	if len(history) > 4 {
		start = len(history) - 4
	}
	for _, msg := range history[start:] {
		fmt.Fprintf(&historyStr, "%s: %s\n", strings.Title(msg.Role), msg.Content)
	}

	system := fmt.Sprintf(`You are a writing assistant for the story "%s". Answer questions using ONLY the provided scene excerpts. Be specific. After your answer, write "Sources:" on a new line, then list the scene titles you drew from, one per line.`, projectName)

	user := fmt.Sprintf(`%sScene excerpts:
%s

%sQuestion: %s`, historyStr.String(), context.String(), "", question)

	return s.call(system, user, 0.4)
}

// ── JSON extraction helper ─────────────────────────────────────────────────────

// extractJSON strips markdown fences and finds the outermost JSON object.
func extractJSON(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		parts := strings.SplitN(raw, "\n", 2)
		if len(parts) > 1 {
			raw = parts[1]
		}
		if idx := strings.LastIndex(raw, "```"); idx >= 0 {
			raw = strings.TrimSpace(raw[:idx])
		}
	}
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		return raw[start : end+1]
	}
	return raw
}
