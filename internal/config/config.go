package config

import (
	"os"
	"strconv"
	"time"
)

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
	InferenceSteps    int           // ACE-Step turbo steps
}

// Load reads configuration from environment variables with sane defaults.
func Load() Config {
	return Config{
		ACEStepAPIURL:    envStr("ACESTEP_API_URL", "http://acestep:8000"),
		ACEStepAPIKey:    envStr("ACESTEP_API_KEY", ""),
		ACEStepOutputDir: envStr("ACESTEP_OUTPUT_DIR", "/acestep-outputs"),

		Port: envInt("RADIO_PORT", 8080),

		StartingGenre:     envStr("RADIO_GENRE", "lofi hip hop"),
		TrackDuration:     envInt("RADIO_TRACK_DURATION", 180),
		CrossfadeDuration: time.Duration(envInt("RADIO_CROSSFADE_DURATION", 8)) * time.Second,
		BufferAhead:       envInt("RADIO_BUFFER_AHEAD", 3),
		DwellMin:          envInt("RADIO_DWELL_MIN", 300),
		DwellMax:          envInt("RADIO_DWELL_MAX", 900),
		InferenceSteps:    envInt("RADIO_INFERENCE_STEPS", 8),
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
