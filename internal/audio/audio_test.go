package audio

import (
	"testing"
	"time"
)

// --- Constants ---

func TestConstants(t *testing.T) {
	// 48kHz * 20ms = 960 samples per channel
	if got := SampleRate * int(FrameDuration/time.Millisecond) / 1000; got != FrameSize {
		t.Errorf("FrameSize mismatch: want %d, got %d", got, FrameSize)
	}
	if FrameSamples != FrameSize*Channels {
		t.Errorf("FrameSamples = %d, want %d", FrameSamples, FrameSize*Channels)
	}
	if FrameBytes != FrameSamples*2 {
		t.Errorf("FrameBytes = %d, want %d", FrameBytes, FrameSamples*2)
	}
}

// --- Smoothstep ---

func TestSmoothstepBoundaries(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{-0.5, 0},
		{0, 0},
		{0.5, 0.5},
		{1, 1},
		{1.5, 1},
	}
	for _, tt := range tests {
		got := Smoothstep(tt.input)
		if got != tt.want {
			t.Errorf("Smoothstep(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSmoothstepMonotonic(t *testing.T) {
	prev := 0.0
	for i := 1; i <= 100; i++ {
		x := float64(i) / 100.0
		val := Smoothstep(x)
		if val < prev {
			t.Errorf("Smoothstep not monotonic: f(%v)=%v < f(%v)=%v", x, val, float64(i-1)/100.0, prev)
		}
		prev = val
	}
}

func TestSmoothstepSymmetry(t *testing.T) {
	// Smoothstep is symmetric around 0.5: f(0.5+d) + f(0.5-d) = 1
	for _, d := range []float64{0.1, 0.2, 0.3, 0.4, 0.5} {
		sum := Smoothstep(0.5+d) + Smoothstep(0.5-d)
		if diff := sum - 1.0; diff > 1e-10 || diff < -1e-10 {
			t.Errorf("Smoothstep symmetry broken at d=%v: sum=%v", d, sum)
		}
	}
}

// --- CrossfadeFrames ---

func TestCrossfadeAllOutgoing(t *testing.T) {
	out := []int16{1000, -1000, 500, -500}
	in := []int16{2000, -2000, 1500, -1500}
	result := CrossfadeFrames(out, in, 0)
	for i, v := range result {
		if v != out[i] {
			t.Errorf("At progress=0 sample[%d] = %d, want %d (all outgoing)", i, v, out[i])
		}
	}
}

func TestCrossfadeAllIncoming(t *testing.T) {
	out := []int16{1000, -1000, 500, -500}
	in := []int16{2000, -2000, 1500, -1500}
	result := CrossfadeFrames(out, in, 1)
	for i, v := range result {
		if v != in[i] {
			t.Errorf("At progress=1 sample[%d] = %d, want %d (all incoming)", i, v, in[i])
		}
	}
}

func TestCrossfadeMidpoint(t *testing.T) {
	out := []int16{1000, -1000}
	in := []int16{3000, -3000}
	result := CrossfadeFrames(out, in, 0.5)
	// At midpoint, smoothstep(0.5)=0.5, so average: (1000*0.5 + 3000*0.5) = 2000
	for i, want := range []int16{2000, -2000} {
		if result[i] != want {
			t.Errorf("At progress=0.5 sample[%d] = %d, want %d", i, result[i], want)
		}
	}
}

func TestCrossfadeClipping(t *testing.T) {
	out := []int16{32000, -32000}
	in := []int16{32000, -32000}
	result := CrossfadeFrames(out, in, 0.5)
	// Both loud at midpoint: 32000*0.5 + 32000*0.5 = 32000 (no clipping needed here)
	// But test with values that would overflow:
	out2 := []int16{32767, -32768}
	in2 := []int16{32767, -32768}
	result2 := CrossfadeFrames(out2, in2, 0.5)
	if result[0] > 32767 || result[0] < -32768 {
		t.Errorf("Clipping failed: got %d", result[0])
	}
	if result2[0] != 32767 {
		t.Errorf("Max values at midpoint: got %d, want 32767", result2[0])
	}
	if result2[1] != -32768 {
		t.Errorf("Min values at midpoint: got %d, want -32768", result2[1])
	}
}

// --- SamplesToBytes / round-trip ---

func TestSamplesToBytes(t *testing.T) {
	samples := []int16{0, 1, -1, 32767, -32768, 256}
	buf := SamplesToBytes(samples)
	if len(buf) != len(samples)*2 {
		t.Fatalf("SamplesToBytes length = %d, want %d", len(buf), len(samples)*2)
	}

	// Verify little-endian encoding manually for a few values
	// 256 = 0x0100 -> bytes [0x00, 0x01]
	idx := 5 * 2
	if buf[idx] != 0x00 || buf[idx+1] != 0x01 {
		t.Errorf("Sample 256 encoded as [%02x, %02x], want [00, 01]", buf[idx], buf[idx+1])
	}
}

func TestSamplesBytesRoundTrip(t *testing.T) {
	original := []int16{0, 1, -1, 32767, -32768, 12345, -6789}
	buf := SamplesToBytes(original)

	// Decode back
	recovered := make([]int16, len(buf)/2)
	for i := range recovered {
		recovered[i] = int16(uint16(buf[i*2]) | uint16(buf[i*2+1])<<8)
	}

	for i, v := range original {
		if recovered[i] != v {
			t.Errorf("Round-trip sample[%d]: got %d, want %d", i, recovered[i], v)
		}
	}
}

// --- Pipeline unit tests (non-I/O) ---

func TestNewPipeline(t *testing.T) {
	p := NewPipeline(8 * time.Second)
	if p == nil {
		t.Fatal("NewPipeline returned nil")
	}
	if p.crossfadeDur != 8*time.Second {
		t.Errorf("crossfadeDur = %v, want 8s", p.crossfadeDur)
	}
}

func TestPipelineQueueSize(t *testing.T) {
	p := NewPipeline(4 * time.Second)
	if p.QueueSize() != 0 {
		t.Errorf("Initial QueueSize = %d, want 0", p.QueueSize())
	}
}

func TestPipelineStatus(t *testing.T) {
	p := NewPipeline(4 * time.Second)
	track, pos, dur := p.Status()
	if track.ID != "" || pos != 0 || dur != 0 {
		t.Errorf("Initial status should be zero-valued, got track=%v pos=%v dur=%v", track, pos, dur)
	}
}

func TestPipelineSkipNonBlocking(t *testing.T) {
	p := NewPipeline(4 * time.Second)
	// Skip on empty channel should not block
	p.Skip()
	p.Skip() // second skip also shouldn't block (buffered channel of 1, first fills it)
}
