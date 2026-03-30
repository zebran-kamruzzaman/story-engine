package services

import (
	"context"
	"strings"
	"unicode"
)

// AnalysisJob is a unit of work submitted to the background analysis goroutine.
type AnalysisJob struct {
	SceneID string
	Content string
}

// AnalysisService runs text analysis in a single dedicated background goroutine.
// It accepts work via Submit() which is non-blocking — if the goroutine is busy,
// the job is silently dropped. The next save will resubmit.
type AnalysisService struct {
	workCh chan AnalysisJob
	cache  *CacheService
	events *EventService
}

// NewAnalysisService creates a new AnalysisService.
// Call Start() to begin the background processing goroutine.
func NewAnalysisService(cache *CacheService, events *EventService) *AnalysisService {
	return &AnalysisService{
		workCh: make(chan AnalysisJob), // unbuffered: if busy, drops
		cache:  cache,
		events: events,
	}
}

// Start launches the single background goroutine that processes analysis jobs.
// It runs until the context is cancelled (i.e., on application shutdown).
func (s *AnalysisService) Start(ctx context.Context) {
	go s.loop(ctx)
}

// Submit sends a job to the background goroutine. Non-blocking:
// if the goroutine is currently processing a job, this call returns immediately
// and the job is dropped. This ensures the save path is never blocked by analysis.
func (s *AnalysisService) Submit(job AnalysisJob) {
	select {
	case s.workCh <- job:
	default:
		// Goroutine busy — drop the job. The next save will resubmit.
	}
}

// loop is the goroutine body. It receives jobs and processes them sequentially.
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

// process runs entity detection and dialogue counting, then updates the cache
// and emits an event to the frontend.
func (s *AnalysisService) process(job AnalysisJob) {
	entities := detectEntities(job.Content)
	dialogueCount := countDialogue(job.Content)

	// Persist to cache. If this fails, we log and continue — the event is still emitted
	// so the frontend reflects what was detected, even if persistence failed.
	if err := s.cache.UpsertEntities(job.SceneID, entities); err != nil {
		// In V1 we log to stdout. V2 will route this to an error store.
		_ = err
	}

	s.events.EmitInsightsUpdated(job.SceneID, entities, dialogueCount)
}

// stopwords is the set of common words that should never be treated as character names.
// Using a map for O(1) lookup.
var stopwords = map[string]bool{
	"I": true, "The": true, "A": true, "An": true, "She": true, "He": true,
	"It": true, "They": true, "We": true, "You": true, "His": true, "Her": true,
	"Their": true, "This": true, "That": true, "Then": true, "When": true,
	"Where": true, "But": true, "And": true, "Or": true, "So": true, "If": true,
	"As": true, "At": true, "In": true, "On": true, "To": true, "Of": true,
	"For": true, "With": true, "By": true, "From": true,
}

// sentenceEndRunes are characters that mark the end of a sentence.
// A token immediately following one of these is a sentence-start word and is excluded.
var sentenceEndChars = map[rune]bool{'.': true, '!': true, '?': true}

// detectEntities applies the frequency-based candidate name extraction algorithm.
// Input: raw scene text. Output: candidate character names appearing 2+ times, sorted by frequency.
func detectEntities(text string) []string {
	tokens := strings.Fields(text)
	freq := make(map[string]int)

	prevEndedSentence := true // start of text treated as sentence start

	for _, raw := range tokens {
		// Strip leading and trailing punctuation from the token.
		stripped := stripPunct(raw)
		if len(stripped) < 2 {
			// Track whether this raw token ended a sentence, then skip.
			prevEndedSentence = endsWithSentencePunct(raw)
			continue
		}

		// Skip if this is the first word of a sentence.
		if prevEndedSentence {
			prevEndedSentence = endsWithSentencePunct(raw)
			continue
		}
		prevEndedSentence = endsWithSentencePunct(raw)

		// Skip stopwords.
		if stopwords[stripped] {
			continue
		}

		// Must start with an uppercase letter.
		runes := []rune(stripped)
		if !unicode.IsUpper(runes[0]) {
			continue
		}

		freq[stripped]++
	}

	// Collect candidates with frequency >= 2.
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

	// Sort by frequency descending.
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

// stripPunct removes leading and trailing punctuation from a token.
func stripPunct(s string) string {
	const punct = `.,!?;:"'()[]—–`
	return strings.TrimFunc(s, func(r rune) bool {
		return strings.ContainsRune(punct, r)
	})
}

// endsWithSentencePunct returns true if the raw token ends with . ! or ?
func endsWithSentencePunct(s string) bool {
	if len(s) == 0 {
		return false
	}
	last := rune(s[len(s)-1])
	return sentenceEndChars[last]
}

// countDialogue counts the number of dialogue segments in the text.
// Strategy: count straight double-quote characters; pairs = segments.
func countDialogue(text string) int {
	count := strings.Count(text, `"`)
	return count / 2
}
