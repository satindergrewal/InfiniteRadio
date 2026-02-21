package config

import (
	"os"
	"strconv"
	"time"
)

func envFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// Config holds all runtime configuration, loaded from environment variables.
type Config struct {
	// ACE-Step connection
	ACEStepAPIURL    string
	ACEStepAPIKey    string
	ACEStepOutputDir string

	// Server
	Port int

	// Radio behavior
	StartingGenre     string
	TrackDuration     int           // seconds
	CrossfadeDuration time.Duration // crossfade length
	BufferAhead       int           // tracks to pre-generate
	DwellMin          int           // min seconds per genre
	DwellMax          int           // max seconds per genre

	// ACE-Step generation quality
	InferenceSteps int     // diffusion steps (base model: 50+, turbo: 8)
	GuidanceScale  float64 // CFG strength (base/sft only, 4.0 is sweet spot)
	Shift          float64 // timestep shift (1.0-5.0, base model only)
	AudioFormat    string  // output format: flac, mp3, wav
}

// Load reads configuration from environment variables with sane defaults.
func Load() Config {
	return Config{
		ACEStepAPIURL:    envStr("ACESTEP_API_URL", "http://acestep:8000"),
		ACEStepAPIKey:    envStr("ACESTEP_API_KEY", ""),
		ACEStepOutputDir: envStr("ACESTEP_OUTPUT_DIR", "/acestep-outputs"),

		Port: envInt("RADIO_PORT", 8080),

		StartingGenre:     envStr("RADIO_GENRE", "lofi hip hop"),
		TrackDuration:     envInt("RADIO_TRACK_DURATION", 90),
		CrossfadeDuration: time.Duration(envInt("RADIO_CROSSFADE_DURATION", 8)) * time.Second,
		BufferAhead:       envInt("RADIO_BUFFER_AHEAD", 3),
		DwellMin:          envInt("RADIO_DWELL_MIN", 300),
		DwellMax:          envInt("RADIO_DWELL_MAX", 900),
		InferenceSteps:    envInt("RADIO_INFERENCE_STEPS", 50),
		GuidanceScale:     envFloat("RADIO_GUIDANCE_SCALE", 4.0),
		Shift:             envFloat("RADIO_SHIFT", 3.0),
		AudioFormat:       envStr("RADIO_AUDIO_FORMAT", "flac"),
	}
}

func envStr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
