package audio

import "time"

const (
	SampleRate    = 48000
	Channels      = 2
	BitDepth      = 16
	FrameDuration = 20 * time.Millisecond
	FrameSize     = 960                    // samples per channel per 20ms frame
	FrameSamples  = FrameSize * Channels   // total interleaved samples per frame
	FrameBytes    = FrameSamples * 2       // bytes per frame (int16 = 2 bytes)
)

// TrackInfo identifies a generated track for the pipeline.
type TrackInfo struct {
	ID    string
	Genre string
	Path  string
	Name  string // display name (LLM-generated or deterministic)
}
