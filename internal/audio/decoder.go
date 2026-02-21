package audio

import (
	"encoding/binary"
	"fmt"
	"os/exec"
)

// DecodeFile runs FFmpeg to decode an audio file to raw PCM int16 samples.
// Returns interleaved stereo samples at 48kHz.
func DecodeFile(path string) ([]int16, error) {
	cmd := exec.Command("ffmpeg",
		"-i", path,
		"-f", "s16le",
		"-acodec", "pcm_s16le",
		"-ar", "48000",
		"-ac", "2",
		"-loglevel", "error",
		"pipe:1",
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffmpeg decode %s: %w", path, err)
	}

	// Ensure even byte count for int16 alignment
	if len(out)%2 != 0 {
		out = out[:len(out)-1]
	}

	samples := make([]int16, len(out)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(out[i*2 : i*2+2]))
	}

	return samples, nil
}

// SamplesToBytes converts int16 samples to little-endian bytes.
func SamplesToBytes(samples []int16) []byte {
	buf := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
	}
	return buf
}
