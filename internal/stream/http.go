package stream

import (
	"context"
	"io"
	"log"
	"net/http"
	"os/exec"

	"github.com/satindergrewal/drift/internal/audio"
)

// HTTPHandler serves a chunked MP3 audio stream via HTTP.
// Each connection spawns an FFmpeg process to encode PCM -> MP3 in real-time.
type HTTPHandler struct {
	broadcaster *Broadcaster
}

// NewHTTPHandler creates an HTTP stream handler.
func NewHTTPHandler(b *Broadcaster) *HTTPHandler {
	return &HTTPHandler{broadcaster: b}
}

func (h *HTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache, no-store")
	w.Header().Set("Connection", "close")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("ICY-Name", "drift radio")

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// FFmpeg: PCM stdin -> MP3 stdout
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-f", "s16le",
		"-ar", "48000",
		"-ac", "2",
		"-i", "pipe:0",
		"-codec:a", "libmp3lame",
		"-b:a", "192k",
		"-f", "mp3",
		"-fflags", "nobuffer",
		"-flush_packets", "1",
		"-loglevel", "error",
		"pipe:1",
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("HTTP stream: stdin pipe error: %v", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("HTTP stream: stdout pipe error: %v", err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("HTTP stream: ffmpeg start error: %v", err)
		return
	}

	listener := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(listener)

	log.Printf("HTTP listener connected (total: %d)", h.broadcaster.ListenerCount())
	defer log.Printf("HTTP listener disconnected")

	// Feed PCM frames to FFmpeg
	go func() {
		defer stdin.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case <-listener.done:
				return
			case frame, ok := <-listener.C:
				if !ok {
					return
				}
				pcm := audio.SamplesToBytes(frame)
				if _, err := stdin.Write(pcm); err != nil {
					return
				}
			}
		}
	}()

	// Read MP3 from FFmpeg and write to HTTP response
	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				break
			}
			flusher.Flush()
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("HTTP stream: ffmpeg read error: %v", err)
			}
			break
		}
	}

	cmd.Wait()
}
