package autodj

import (
	"context"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/satindergrewal/drift/internal/acestep"
	"github.com/satindergrewal/drift/internal/audio"
)

// SchedulerConfig holds auto-DJ parameters.
type SchedulerConfig struct {
	StartingGenre  string
	TrackDuration  int     // seconds
	BufferAhead    int     // tracks to pre-generate
	DwellMin       int     // min seconds per genre
	DwellMax       int     // max seconds per genre
	InferenceSteps int     // diffusion steps (base: 50+, turbo: 8)
	GuidanceScale  float64 // CFG strength (base/sft only)
	Shift          float64 // timestep shift
	AudioFormat    string  // flac, mp3, wav
}

// SchedulerStatus is the current state of the auto-DJ.
type SchedulerStatus struct {
	CurrentGenre   string  `json:"genre"`
	AutoDJ         bool    `json:"auto_dj"`
	DwellRemaining float64 `json:"dwell_remaining"` // seconds
	QueueSize      int     `json:"queue_size"`
}

// CaptionFunc generates a caption for a genre. Returns empty string on failure.
type CaptionFunc func(ctx context.Context, genre string) string

// NameFunc generates a track name from genre, trackID, and caption.
// Returns empty string on failure.
type NameFunc func(ctx context.Context, genre, trackID, caption string) string

// Scheduler manages genre transitions and track generation.
type Scheduler struct {
	client   *acestep.Client
	pipeline *audio.Pipeline
	cfg      SchedulerConfig

	captionFn CaptionFunc // optional LLM caption generator
	nameFn    NameFunc    // optional LLM track name generator

	mu           sync.RWMutex
	currentGenre string
	autoDJ       bool
	dwellEnd     time.Time
	lastCaption  string // last generated caption (for status display)

	genreOverrideCh chan string
}

// NewScheduler creates an auto-DJ scheduler.
func NewScheduler(client *acestep.Client, pipeline *audio.Pipeline, cfg SchedulerConfig) *Scheduler {
	return &Scheduler{
		client:          client,
		pipeline:        pipeline,
		cfg:             cfg,
		currentGenre:    cfg.StartingGenre,
		autoDJ:          true,
		genreOverrideCh: make(chan string, 1),
	}
}

// SetCaptionFunc sets the LLM-powered caption generator. Pass nil to use static captions.
func (s *Scheduler) SetCaptionFunc(fn CaptionFunc) {
	s.mu.Lock()
	s.captionFn = fn
	s.mu.Unlock()
}

// SetNameFunc sets the LLM-powered name generator. Pass nil to use deterministic names.
func (s *Scheduler) SetNameFunc(fn NameFunc) {
	s.mu.Lock()
	s.nameFn = fn
	s.mu.Unlock()
}

// LastCaption returns the caption used for the most recent track generation.
func (s *Scheduler) LastCaption() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastCaption
}

// Status returns the current DJ state.
func (s *Scheduler) Status() SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	remaining := time.Until(s.dwellEnd).Seconds()
	if remaining < 0 {
		remaining = 0
	}
	return SchedulerStatus{
		CurrentGenre:   s.currentGenre,
		AutoDJ:         s.autoDJ,
		DwellRemaining: remaining,
		QueueSize:      s.pipeline.QueueSize(),
	}
}

// SetGenre manually overrides the current genre.
func (s *Scheduler) SetGenre(genre string) {
	select {
	case s.genreOverrideCh <- genre:
	default:
	}
}

// Skip skips the current track.
func (s *Scheduler) Skip() {
	s.pipeline.Skip()
}

// SetAutoDJ enables or disables automatic genre transitions.
func (s *Scheduler) SetAutoDJ(enabled bool) {
	s.mu.Lock()
	s.autoDJ = enabled
	if enabled {
		s.resetDwell()
	}
	s.mu.Unlock()
}

// SetTrackDuration updates the duration for future generated tracks (seconds).
func (s *Scheduler) SetTrackDuration(seconds int) {
	s.mu.Lock()
	s.cfg.TrackDuration = seconds
	s.mu.Unlock()
	log.Printf("Track duration set to %ds", seconds)
}

// TrackDuration returns the current track duration setting.
func (s *Scheduler) TrackDuration() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.TrackDuration
}

// Run starts the auto-DJ loop. Blocks until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	s.mu.Lock()
	s.resetDwell()
	s.mu.Unlock()

	log.Printf("Auto-DJ started with genre: %s", s.cfg.StartingGenre)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Check for manual genre override
		select {
		case genre := <-s.genreOverrideCh:
			s.mu.Lock()
			s.currentGenre = genre
			s.resetDwell()
			s.mu.Unlock()
			log.Printf("Genre manually set to: %s", genre)
		default:
		}

		// Check dwell expiry for auto-transition
		s.mu.RLock()
		autoDJ := s.autoDJ
		expired := time.Now().After(s.dwellEnd)
		s.mu.RUnlock()

		if autoDJ && expired {
			s.transitionGenre()
		}

		// Keep the generation buffer full
		if s.pipeline.QueueSize() < s.cfg.BufferAhead {
			s.generateTrack(ctx)
		} else {
			time.Sleep(time.Second)
		}
	}
}

func (s *Scheduler) generateTrack(ctx context.Context) {
	s.mu.RLock()
	genre := s.currentGenre
	trackDur := s.cfg.TrackDuration
	captionFn := s.captionFn
	nameFn := s.nameFn
	s.mu.RUnlock()

	// Try LLM caption first, fall back to static.
	// Use a short timeout so a slow LLM never blocks track generation.
	var caption string
	if captionFn != nil {
		llmCtx, llmCancel := context.WithTimeout(ctx, 15*time.Second)
		caption = captionFn(llmCtx, genre)
		llmCancel()
	}
	if caption == "" {
		caption = GetCaption(genre)
	}

	s.mu.Lock()
	s.lastCaption = caption
	s.mu.Unlock()

	log.Printf("Generating %s track...", genre)

	taskID, err := s.client.Generate(ctx, acestep.GenerateRequest{
		Caption:        caption,
		Lyrics:         "[Instrumental]",
		Duration:       trackDur,
		InferenceSteps: s.cfg.InferenceSteps,
		GuidanceScale:  s.cfg.GuidanceScale,
		Shift:          s.cfg.Shift,
		InferMethod:    "ode",
		Thinking:       true,
		UseCotCaption:  true,
		UseCotLanguage: true,
		VocalLanguage:  "en",
		Seed:           -1,
		UseRandomSeed:  true,
		BatchSize:      1,
		AudioFormat:    s.cfg.AudioFormat,
	})
	if err != nil {
		log.Printf("Generate error: %v", err)
		time.Sleep(5 * time.Second)
		return
	}

	path, err := s.client.PollUntilDone(ctx, taskID, 3*time.Second)
	if err != nil {
		log.Printf("Poll error for task %s: %v", taskID, err)
		return
	}

	// Try LLM name, fall back to deterministic.
	var trackName string
	if nameFn != nil {
		nameCtx, nameCancel := context.WithTimeout(ctx, 15*time.Second)
		trackName = nameFn(nameCtx, genre, taskID, caption)
		nameCancel()
	}
	if trackName == "" {
		trackName = TrackName(genre, taskID)
	}

	log.Printf("Track ready: %s [%s] (genre: %s)", trackName, taskID, genre)

	s.pipeline.Enqueue(audio.TrackInfo{
		ID:    taskID,
		Genre: genre,
		Path:  path,
		Name:  trackName,
	})
}

func (s *Scheduler) transitionGenre() {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := MoodGraph[s.currentGenre]
	if !ok || len(g.Adjacent) == 0 {
		s.resetDwell()
		return
	}

	next := g.Adjacent[rand.IntN(len(g.Adjacent))]
	log.Printf("Auto-DJ transition: %s -> %s", s.currentGenre, next)
	s.currentGenre = next
	s.resetDwell()
}

// resetDwell sets a new random dwell timer. Must be called with mu held.
func (s *Scheduler) resetDwell() {
	spread := s.cfg.DwellMax - s.cfg.DwellMin
	if spread <= 0 {
		spread = 1
	}
	dwell := s.cfg.DwellMin + rand.IntN(spread)
	s.dwellEnd = time.Now().Add(time.Duration(dwell) * time.Second)
}
