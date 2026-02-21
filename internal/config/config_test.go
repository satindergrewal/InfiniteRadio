package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any env vars that might interfere
	envVars := []string{
		"ACESTEP_API_URL", "ACESTEP_API_KEY", "ACESTEP_OUTPUT_DIR",
		"RADIO_PORT", "RADIO_GENRE", "RADIO_TRACK_DURATION",
		"RADIO_CROSSFADE_DURATION", "RADIO_BUFFER_AHEAD",
		"RADIO_DWELL_MIN", "RADIO_DWELL_MAX", "RADIO_INFERENCE_STEPS",
		"RADIO_GUIDANCE_SCALE", "RADIO_SHIFT", "RADIO_AUDIO_FORMAT",
	}
	for _, k := range envVars {
		os.Unsetenv(k)
	}

	cfg := Load()

	if cfg.ACEStepAPIURL != "http://acestep:8000" {
		t.Errorf("ACEStepAPIURL = %q, want default", cfg.ACEStepAPIURL)
	}
	if cfg.ACEStepAPIKey != "" {
		t.Errorf("ACEStepAPIKey = %q, want empty default", cfg.ACEStepAPIKey)
	}
	if cfg.ACEStepOutputDir != "/acestep-outputs" {
		t.Errorf("ACEStepOutputDir = %q, want default", cfg.ACEStepOutputDir)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.StartingGenre != "lofi hip hop" {
		t.Errorf("StartingGenre = %q, want 'lofi hip hop'", cfg.StartingGenre)
	}
	if cfg.TrackDuration != 90 {
		t.Errorf("TrackDuration = %d, want 90", cfg.TrackDuration)
	}
	if cfg.CrossfadeDuration != 8*time.Second {
		t.Errorf("CrossfadeDuration = %v, want 8s", cfg.CrossfadeDuration)
	}
	if cfg.BufferAhead != 3 {
		t.Errorf("BufferAhead = %d, want 3", cfg.BufferAhead)
	}
	if cfg.DwellMin != 300 {
		t.Errorf("DwellMin = %d, want 300", cfg.DwellMin)
	}
	if cfg.DwellMax != 900 {
		t.Errorf("DwellMax = %d, want 900", cfg.DwellMax)
	}
	if cfg.InferenceSteps != 50 {
		t.Errorf("InferenceSteps = %d, want 50", cfg.InferenceSteps)
	}
	if cfg.GuidanceScale != 4.0 {
		t.Errorf("GuidanceScale = %f, want 4.0", cfg.GuidanceScale)
	}
	if cfg.Shift != 3.0 {
		t.Errorf("Shift = %f, want 3.0", cfg.Shift)
	}
	if cfg.AudioFormat != "flac" {
		t.Errorf("AudioFormat = %q, want 'flac'", cfg.AudioFormat)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("ACESTEP_API_URL", "http://localhost:9000")
	t.Setenv("ACESTEP_API_KEY", "test-key-123")
	t.Setenv("ACESTEP_OUTPUT_DIR", "/tmp/outputs")
	t.Setenv("RADIO_PORT", "3000")
	t.Setenv("RADIO_GENRE", "jazz")
	t.Setenv("RADIO_TRACK_DURATION", "60")
	t.Setenv("RADIO_CROSSFADE_DURATION", "4")
	t.Setenv("RADIO_BUFFER_AHEAD", "5")
	t.Setenv("RADIO_DWELL_MIN", "120")
	t.Setenv("RADIO_DWELL_MAX", "600")
	t.Setenv("RADIO_INFERENCE_STEPS", "16")
	t.Setenv("RADIO_GUIDANCE_SCALE", "7.5")
	t.Setenv("RADIO_SHIFT", "4.0")
	t.Setenv("RADIO_AUDIO_FORMAT", "wav")

	cfg := Load()

	if cfg.ACEStepAPIURL != "http://localhost:9000" {
		t.Errorf("ACEStepAPIURL = %q, want env override", cfg.ACEStepAPIURL)
	}
	if cfg.ACEStepAPIKey != "test-key-123" {
		t.Errorf("ACEStepAPIKey = %q, want env override", cfg.ACEStepAPIKey)
	}
	if cfg.ACEStepOutputDir != "/tmp/outputs" {
		t.Errorf("ACEStepOutputDir = %q, want env override", cfg.ACEStepOutputDir)
	}
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000", cfg.Port)
	}
	if cfg.StartingGenre != "jazz" {
		t.Errorf("StartingGenre = %q, want 'jazz'", cfg.StartingGenre)
	}
	if cfg.TrackDuration != 60 {
		t.Errorf("TrackDuration = %d, want 60", cfg.TrackDuration)
	}
	if cfg.CrossfadeDuration != 4*time.Second {
		t.Errorf("CrossfadeDuration = %v, want 4s", cfg.CrossfadeDuration)
	}
	if cfg.BufferAhead != 5 {
		t.Errorf("BufferAhead = %d, want 5", cfg.BufferAhead)
	}
	if cfg.DwellMin != 120 {
		t.Errorf("DwellMin = %d, want 120", cfg.DwellMin)
	}
	if cfg.DwellMax != 600 {
		t.Errorf("DwellMax = %d, want 600", cfg.DwellMax)
	}
	if cfg.InferenceSteps != 16 {
		t.Errorf("InferenceSteps = %d, want 16", cfg.InferenceSteps)
	}
	if cfg.GuidanceScale != 7.5 {
		t.Errorf("GuidanceScale = %f, want 7.5", cfg.GuidanceScale)
	}
	if cfg.Shift != 4.0 {
		t.Errorf("Shift = %f, want 4.0", cfg.Shift)
	}
	if cfg.AudioFormat != "wav" {
		t.Errorf("AudioFormat = %q, want 'wav'", cfg.AudioFormat)
	}
}

func TestEnvIntInvalidFallsBack(t *testing.T) {
	t.Setenv("RADIO_PORT", "not-a-number")
	cfg := Load()
	if cfg.Port != 8080 {
		t.Errorf("Invalid int env should fallback to default: got %d, want 8080", cfg.Port)
	}
}

func TestEnvStrEmpty(t *testing.T) {
	// Empty string should use fallback
	os.Unsetenv("ACESTEP_API_URL")
	cfg := Load()
	if cfg.ACEStepAPIURL != "http://acestep:8000" {
		t.Errorf("Unset env should use fallback: got %q", cfg.ACEStepAPIURL)
	}
}
