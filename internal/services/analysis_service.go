package services

import (
	"context"
	"strings"
	"unicode"
)

// AnalysisJob is a unit of work for the background goroutine.
type AnalysisJob struct {
	SceneID string
	Content string
}

// AnalysisService runs fast rule-based analysis in a background goroutine.
// It runs on every save. Deep LLM analysis is handled by app.go's AnalyzeScene IPC method.
type AnalysisService struct {
	workCh chan AnalysisJob
	cache  *CacheService
	events *EventService
}

func NewAnalysisService(cache *CacheService, events *EventService) *AnalysisService {
	return &AnalysisService{
		workCh: make(chan AnalysisJob),
		cache:  cache,
		events: events,
	}
}

func (s *AnalysisService) Start(ctx context.Context) {
	go s.loop(ctx)
}

func (s *AnalysisService) Submit(job AnalysisJob) {
	select {
	case s.workCh <- job:
	default:
	}
}

func (s *AnalysisService) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-s.workCh:
			s.process(job)
		}
	}
}

// process runs character detection and emits a rule-based mirror update.
// Interactions and SceneTone are empty in rule-based results — only the
// LLM path fills those fields.
func (s *AnalysisService) process(job AnalysisJob) {
	entities := detectEntities(job.Content)

	if err := s.cache.UpsertEntities(job.SceneID, entities); err != nil {
		_ = err
	}

	s.events.EmitMirrorUpdated(job.SceneID, entities, nil, "", "rule")
}

// ─────────────────────────────────────────────────────────────────────────────
// Rule-based character detection (unchanged from V1)
// ─────────────────────────────────────────────────────────────────────────────

var stopwords = map[string]bool{
	"I": true, "The": true, "A": true, "An": true, "She": true, "He": true,
	"It": true, "They": true, "We": true, "You": true, "His": true, "Her": true,
	"Their": true, "This": true, "That": true, "Then": true, "When": true,
	"Where": true, "But": true, "And": true, "Or": true, "So": true, "If": true,
	"As": true, "At": true, "In": true, "On": true, "To": true, "Of": true,
	"For": true, "With": true, "By": true, "From": true,
}

var sentenceEndChars = map[rune]bool{'.': true, '!': true, '?': true}

func detectEntities(text string) []string {
	tokens := strings.Fields(text)
	freq := make(map[string]int)
	prevEndedSentence := true

	for _, raw := range tokens {
		stripped := stripPunct(raw)
		if len(stripped) < 2 {
			prevEndedSentence = endsWithSentencePunct(raw)
			continue
		}
		if prevEndedSentence {
			prevEndedSentence = endsWithSentencePunct(raw)
			continue
		}
		prevEndedSentence = endsWithSentencePunct(raw)
		if stopwords[stripped] {
			continue
		}
		runes := []rune(stripped)
		if !unicode.IsUpper(runes[0]) {
			continue
		}
		freq[stripped]++
	}

	type candidate struct {
		name  string
		count int
	}
	var candidates []candidate
	for name, count := range freq {
		if count >= 2 {
			candidates = append(candidates, candidate{name, count})
		}
	}
	for i := 0; i < len(candidates); i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].count > candidates[i].count {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}
	names := make([]string, len(candidates))
	for i, c := range candidates {
		names[i] = c.name
	}
	return names
}

func stripPunct(s string) string {
	const punct = `.,!?;:"'()[]—–`
	return strings.TrimFunc(s, func(r rune) bool {
		return strings.ContainsRune(punct, r)
	})
}

func endsWithSentencePunct(s string) bool {
	if len(s) == 0 {
		return false
	}
	return sentenceEndChars[rune(s[len(s)-1])]
}
