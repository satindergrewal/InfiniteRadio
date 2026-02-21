package ollama

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
)

// CaptionGenerator uses an LLM to create unique ACE-Step captions per track.
type CaptionGenerator struct {
	client *Client

	mu          sync.Mutex
	lastCaption map[string]string // genre -> last caption used (avoid repeats)
}

// NewCaptionGenerator creates a caption generator backed by an Ollama client.
func NewCaptionGenerator(client *Client) *CaptionGenerator {
	return &CaptionGenerator{
		client:      client,
		lastCaption: make(map[string]string),
	}
}

// captionSystemPrompt instructs the LLM to generate ACE-Step captions.
// Drawn from ACE-Step v1.5 documentation best practices and community findings.
const captionSystemPrompt = `You are a music production caption generator for an AI music model called ACE-Step.

Your job: given a genre, output ONE caption of 20-40 words that describes an instrumental track.

Caption rules (from ACE-Step documentation):
- Describe the SOUND, not a story. Focus on: instruments, timbre, effects, tempo, mood, production style.
- Be SPECIFIC: "warm Rhodes piano with gentle chorus effect" not just "piano"
- Name real instruments, effects, and techniques: "fingerpicked nylon guitar", "sidechain compression", "tape saturation", "spring reverb", "808 sub bass"
- Include tempo guidance: use BPM numbers (e.g. "72 BPM") or tempo words ("slow waltz", "uptempo groove")
- Include mood/atmosphere: "late night", "sunrise", "melancholic", "euphoric", "meditative"
- Reference production eras or styles when relevant: "70s analog warmth", "modern crisp mix", "lo-fi bedroom production"
- Vary the instrumentation: don't always use the same instruments for a genre
- Each caption MUST be meaningfully different from any previous caption

NEVER include:
- Lyrics, vocals, singing, or voice references (these are instrumentals)
- Song titles, artist names, or album references
- Explanations, preambles, quotes, or formatting
- The word "instrumental" (it's implied)

Output format: ONLY the caption text. Nothing else. No quotes. No bullet points. No "Here's a caption:". Just the raw caption.`

// GenerateCaption creates a unique ACE-Step caption for a genre.
// Returns empty string on failure (caller should fall back to static caption).
func (g *CaptionGenerator) GenerateCaption(ctx context.Context, genre string) string {
	g.mu.Lock()
	lastCaption := g.lastCaption[genre]
	g.mu.Unlock()

	prompt := fmt.Sprintf("Genre: %s", genre)
	if lastCaption != "" {
		prompt += fmt.Sprintf("\nPrevious caption (do NOT repeat this): %s", lastCaption)
	}

	caption, err := g.client.Generate(ctx, captionSystemPrompt, prompt)
	if err != nil {
		log.Printf("Ollama caption generation failed: %v", err)
		return ""
	}

	// Clean up: remove quotes, trim, strip thinking tags if present
	caption = cleanCaption(caption)

	if caption == "" || len(caption) < 15 {
		log.Printf("Ollama returned unusable caption: %q", caption)
		return ""
	}

	g.mu.Lock()
	g.lastCaption[genre] = caption
	g.mu.Unlock()

	log.Printf("LLM caption [%s]: %s", genre, caption)
	return caption
}

// nameSystemPrompt instructs the LLM to generate evocative track names.
const nameSystemPrompt = `You are a track name generator for an AI radio station.

Given a genre and a music caption, generate a short evocative track name (2-4 words).

Rules:
- Names should feel like real instrumental track titles
- Evocative and atmospheric, not literal
- No genre name in the title (don't say "Jazz Ballad" for jazz)
- No numbers, no "Track 1", no "Untitled"
- Lowercase only

Output ONLY the track name. Nothing else.`

// GenerateName creates an evocative track name from genre and caption.
// Returns empty string on failure (caller should fall back to deterministic name).
func (g *CaptionGenerator) GenerateName(ctx context.Context, genre, caption string) string {
	prompt := fmt.Sprintf("Genre: %s\nCaption: %s", genre, caption)

	name, err := g.client.Generate(ctx, nameSystemPrompt, prompt)
	if err != nil {
		log.Printf("Ollama name generation failed: %v", err)
		return ""
	}

	name = cleanCaption(name)
	name = strings.ToLower(name)

	// Sanity check: should be short
	if name == "" || len(name) > 60 || strings.Count(name, " ") > 4 {
		log.Printf("Ollama returned unusable name: %q", name)
		return ""
	}

	return name
}

// cleanCaption strips common LLM artifacts from output.
func cleanCaption(s string) string {
	s = strings.TrimSpace(s)

	// Strip thinking tags (Qwen 3 thinking mode leakage)
	if idx := strings.Index(s, "</think>"); idx >= 0 {
		s = s[idx+len("</think>"):]
		s = strings.TrimSpace(s)
	}
	// If it starts with <think>, strip everything up to </think>
	if strings.HasPrefix(s, "<think>") {
		if idx := strings.Index(s, "</think>"); idx >= 0 {
			s = s[idx+len("</think>"):]
		}
		s = strings.TrimSpace(s)
	}

	// Strip surrounding quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	// Strip common preambles
	prefixes := []string{
		"Here's a caption:",
		"Here is a caption:",
		"Caption:",
		"Here's the caption:",
	}
	lower := strings.ToLower(s)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, strings.ToLower(p)) {
			s = strings.TrimSpace(s[len(p):])
		}
	}

	return strings.TrimSpace(s)
}
