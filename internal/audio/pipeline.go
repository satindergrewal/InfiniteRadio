package audio

import (
	"context"
	"log"
	"sync"
	"time"
)

type decodedTrack struct {
	info    TrackInfo
	samples []int16
}

// Pipeline decodes tracks, applies crossfade, and outputs PCM frames at real-time rate.
type Pipeline struct {
	trackCh      chan TrackInfo
	frameCh      chan []int16
	skipCh       chan struct{}
	crossfadeDur time.Duration

	mu            sync.RWMutex
	currentTrack  TrackInfo
	trackPosition time.Duration
	trackDuration time.Duration
}

// NewPipeline creates an audio pipeline with the given crossfade duration.
func NewPipeline(crossfadeDuration time.Duration) *Pipeline {
	return &Pipeline{
		trackCh:      make(chan TrackInfo, 8),
		frameCh:      make(chan []int16, 100),
		skipCh:       make(chan struct{}, 1),
		crossfadeDur: crossfadeDuration,
	}
}

// Frames returns the channel of outgoing PCM frames (20ms each).
func (p *Pipeline) Frames() <-chan []int16 {
	return p.frameCh
}

// Enqueue adds a track to the pipeline's playback queue.
func (p *Pipeline) Enqueue(t TrackInfo) {
	p.trackCh <- t
}

// QueueSize returns the number of tracks waiting in the queue.
func (p *Pipeline) QueueSize() int {
	return len(p.trackCh)
}

// Skip interrupts the current track.
func (p *Pipeline) Skip() {
	select {
	case p.skipCh <- struct{}{}:
	default:
	}
}

// Status returns current playback info.
func (p *Pipeline) Status() (track TrackInfo, position, duration time.Duration) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.currentTrack, p.trackPosition, p.trackDuration
}

// Run starts the pipeline. Blocks until ctx is cancelled.
func (p *Pipeline) Run(ctx context.Context) {
	defer close(p.frameCh)

	ticker := time.NewTicker(FrameDuration)
	defer ticker.Stop()

	// Background decoder: converts file paths to decoded PCM
	decodedCh := make(chan *decodedTrack, 4)
	go func() {
		defer close(decodedCh)
		for {
			select {
			case <-ctx.Done():
				return
			case t, ok := <-p.trackCh:
				if !ok {
					return
				}
				samples, err := DecodeFile(t.Path)
				if err != nil {
					log.Printf("Decode failed %s: %v", t.Path, err)
					continue
				}
				select {
				case decodedCh <- &decodedTrack{info: t, samples: samples}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	// Main playback loop
	var pending *decodedTrack
	var startFrame int

	for {
		var dt *decodedTrack

		if pending != nil {
			dt = pending
			pending = nil
		} else {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-decodedCh:
				if !ok {
					return
				}
				dt = d
				startFrame = 0
			}
		}

		next, nextStart := p.playTrack(ctx, ticker, decodedCh, dt, startFrame)
		if next != nil {
			pending = next
			startFrame = nextStart
		} else {
			startFrame = 0
		}
	}
}

// playTrack plays a decoded track with crossfade into the next one if available.
// Returns the next decoded track and starting frame if a crossfade occurred.
func (p *Pipeline) playTrack(ctx context.Context, ticker *time.Ticker, decodedCh <-chan *decodedTrack, dt *decodedTrack, startFrame int) (*decodedTrack, int) {
	samples := dt.samples
	totalFrames := len(samples) / FrameSamples
	cfFrames := int(p.crossfadeDur.Seconds()) * SampleRate / FrameSize
	if cfFrames > totalFrames/2 {
		cfFrames = totalFrames / 2 // don't crossfade more than half the track
	}
	cfStart := totalFrames - cfFrames

	p.setTrack(dt.info, totalFrames)
	log.Printf("Now playing: %s (genre: %s, frames: %d)", dt.info.ID, dt.info.Genre, totalFrames)

	// Play pre-crossfade frames
	for i := startFrame; i < cfStart; i++ {
		if !p.sendFrame(ctx, ticker, samples[i*FrameSamples:(i+1)*FrameSamples]) {
			return nil, 0
		}
		p.updatePosition(i)
	}

	// Try to get next decoded track for crossfade
	var next *decodedTrack
	select {
	case d := <-decodedCh:
		next = d
	default:
	}

	if next != nil {
		// Crossfade zone: blend outgoing with incoming
		for i := 0; i < cfFrames; i++ {
			outPos := (cfStart + i) * FrameSamples
			inPos := i * FrameSamples

			if outPos+FrameSamples > len(samples) || inPos+FrameSamples > len(next.samples) {
				break
			}

			progress := float64(i) / float64(cfFrames)
			frame := CrossfadeFrames(
				samples[outPos:outPos+FrameSamples],
				next.samples[inPos:inPos+FrameSamples],
				progress,
			)

			if !p.sendFrame(ctx, ticker, frame) {
				return nil, 0
			}
			p.updatePosition(cfStart + i)
		}

		log.Printf("Crossfaded into: %s (genre: %s)", next.info.ID, next.info.Genre)
		return next, cfFrames
	}

	// No next track available: play remaining frames without crossfade
	for i := cfStart; i < totalFrames; i++ {
		if !p.sendFrame(ctx, ticker, samples[i*FrameSamples:(i+1)*FrameSamples]) {
			return nil, 0
		}
		p.updatePosition(i)
	}

	return nil, 0
}

// sendFrame waits for the ticker then sends a frame. Returns false on skip or cancel.
func (p *Pipeline) sendFrame(ctx context.Context, ticker *time.Ticker, frame []int16) bool {
	select {
	case <-ctx.Done():
		return false
	case <-p.skipCh:
		log.Println("Track skipped")
		return false
	case <-ticker.C:
	}

	select {
	case p.frameCh <- frame:
		return true
	case <-ctx.Done():
		return false
	}
}

func (p *Pipeline) setTrack(info TrackInfo, totalFrames int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.currentTrack = info
	p.trackPosition = 0
	p.trackDuration = time.Duration(totalFrames) * FrameDuration
}

func (p *Pipeline) updatePosition(frameIdx int) {
	p.mu.Lock()
	p.trackPosition = time.Duration(frameIdx) * FrameDuration
	p.mu.Unlock()
}
