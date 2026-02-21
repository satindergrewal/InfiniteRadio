package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/satindergrewal/infinara/internal/acestep"
	"github.com/satindergrewal/infinara/internal/audio"
	"github.com/satindergrewal/infinara/internal/autodj"
	"github.com/satindergrewal/infinara/internal/config"
	"github.com/satindergrewal/infinara/internal/ollama"
	"github.com/satindergrewal/infinara/internal/stream"
	"github.com/satindergrewal/infinara/internal/web"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ACE-Step client
	client := acestep.NewClient(cfg.ACEStepAPIURL, cfg.ACEStepAPIKey, cfg.ACEStepOutputDir)

	log.Println("infinara starting up...")

	healthCtx, healthCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer healthCancel()
	if err := client.WaitForHealthy(healthCtx); err != nil {
		log.Fatalf("ACE-Step not available: %v", err)
	}

	// Audio pipeline
	pipeline := audio.NewPipeline(cfg.CrossfadeDuration)
	go pipeline.Run(ctx)

	// Broadcaster: fan-out PCM frames to all listeners
	broadcaster := stream.NewBroadcaster()
	go broadcaster.Run(ctx, pipeline.Frames())

	// Auto-DJ scheduler
	sched := autodj.NewScheduler(client, pipeline, autodj.SchedulerConfig{
		StartingGenre:  cfg.StartingGenre,
		TrackDuration:  cfg.TrackDuration,
		BufferAhead:    cfg.BufferAhead,
		DwellMin:       cfg.DwellMin,
		DwellMax:       cfg.DwellMax,
		InferenceSteps: cfg.InferenceSteps,
		GuidanceScale:  cfg.GuidanceScale,
		Shift:          cfg.Shift,
		AudioFormat:    cfg.AudioFormat,
	})
	// Ollama LLM (optional -- enhances captions and track names)
	var ollamaModel string
	if cfg.OllamaURL != "" {
		ollamaClient := ollama.NewClient(cfg.OllamaURL, cfg.OllamaModel)
		ollamaModel = cfg.OllamaModel

		readyCtx, readyCancel := context.WithTimeout(ctx, 30*time.Second)
		if ollamaClient.WaitForReady(readyCtx) {
			captionGen := ollama.NewCaptionGenerator(ollamaClient)
			sched.SetCaptionFunc(captionGen.GenerateCaption)
			sched.SetNameFunc(func(ctx context.Context, genre, trackID, caption string) string {
				return captionGen.GenerateName(ctx, genre, caption)
			})
			sched.SetStructureFunc(captionGen.GenerateStructure)
			log.Printf("Ollama connected: %s (LLM captions + structure enabled)", cfg.OllamaModel)
		} else {
			log.Println("Ollama not available, using static captions")
		}
		readyCancel()
	} else {
		log.Println("Ollama not configured (set OLLAMA_URL to enable LLM captions)")
	}

	// WebRTC handler (track peer count for status)
	webrtcHandler := stream.NewWebRTCHandler(broadcaster)

	// Idle detection: pause generation when nobody is listening
	sched.SetListenerCountFunc(func() int {
		return broadcaster.ListenerCount() + webrtcHandler.PeerCount()
	})

	go sched.Run(ctx)

	// HTTP routes
	mux := http.NewServeMux()

	// Web UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(web.IndexHTML)
	})

	// Audio streams
	mux.Handle("/stream", stream.NewHTTPHandler(broadcaster))
	mux.Handle("/offer", webrtcHandler)

	// API endpoints
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		djStatus := sched.Status()
		track, pos, dur := pipeline.Status()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		// Use stored name, fall back to deterministic
		trackName := track.Name
		if trackName == "" {
			trackName = autodj.TrackName(track.Genre, track.ID)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"genre":            djStatus.CurrentGenre,
			"auto_dj":          djStatus.AutoDJ,
			"idle":             djStatus.Idle,
			"dwell_remaining":  djStatus.DwellRemaining,
			"queue_size":       djStatus.QueueSize,
			"track_id":         track.ID,
			"track_name":       trackName,
			"track_path":       track.Path,
			"position":         pos.Seconds(),
			"duration":         dur.Seconds(),
			"caption":          sched.LastCaption(),
			"lyrics":           sched.LastLyrics(),
			"http_listeners":   broadcaster.ListenerCount(),
			"webrtc_listeners": webrtcHandler.PeerCount(),
			"config": map[string]any{
				"model":           "acestep-v15-base",
				"inference_steps": cfg.InferenceSteps,
				"guidance_scale":  cfg.GuidanceScale,
				"shift":           cfg.Shift,
				"audio_format":    cfg.AudioFormat,
				"track_duration":  sched.TrackDuration(),
				"crossfade":       pipeline.CrossfadeDuration().Seconds(),
				"llm_model":       ollamaModel,
			},
		})
	})

	mux.HandleFunc("/api/genre", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Genre string `json:"genre"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Genre == "" {
			http.Error(w, "invalid genre", http.StatusBadRequest)
			return
		}
		if !autodj.IsValidGenre(req.Genre) {
			http.Error(w, "unknown genre", http.StatusBadRequest)
			return
		}
		sched.SetGenre(req.Genre)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "genre": req.Genre})
	})

	mux.HandleFunc("/api/skip", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		sched.Skip()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/autodj", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		sched.SetAutoDJ(req.Enabled)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "auto_dj": req.Enabled})
	})

	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		track, _, _ := pipeline.Status()
		if track.Path == "" {
			http.Error(w, "no track playing", http.StatusNotFound)
			return
		}
		saveName := track.Name
		if saveName == "" {
			saveName = autodj.TrackName(track.Genre, track.ID)
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`, saveName, cfg.AudioFormat))
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, track.Path)
	})

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			TrackDuration *int     `json:"track_duration"`
			Crossfade     *float64 `json:"crossfade"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		if req.TrackDuration != nil {
			v := *req.TrackDuration
			if v < 15 || v > 300 {
				http.Error(w, "track_duration must be 15-300", http.StatusBadRequest)
				return
			}
			sched.SetTrackDuration(v)
		}
		if req.Crossfade != nil {
			v := *req.Crossfade
			if v < 1 || v > 30 {
				http.Error(w, "crossfade must be 1-30", http.StatusBadRequest)
				return
			}
			pipeline.SetCrossfade(time.Duration(v * float64(time.Second)))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":             true,
			"track_duration": sched.TrackDuration(),
			"crossfade":      pipeline.CrossfadeDuration().Seconds(),
		})
	})

	mux.HandleFunc("/api/rate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Rating int `json:"rating"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		// Phase 1: store rating for future preference learning
		track, _, _ := pipeline.Status()
		log.Printf("Rating: track=%s genre=%s rating=%d", track.ID, track.Genre, req.Rating)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down...")
		server.Close()
	}()

	log.Printf("infinara live on %s", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
