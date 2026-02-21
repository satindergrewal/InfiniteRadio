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

	"github.com/satindergrewal/drift/internal/acestep"
	"github.com/satindergrewal/drift/internal/audio"
	"github.com/satindergrewal/drift/internal/autodj"
	"github.com/satindergrewal/drift/internal/config"
	"github.com/satindergrewal/drift/internal/stream"
	"github.com/satindergrewal/drift/internal/web"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// ACE-Step client
	client := acestep.NewClient(cfg.ACEStepAPIURL, cfg.ACEStepAPIKey, cfg.ACEStepOutputDir)

	log.Println("drift radio starting up...")

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
	})
	go sched.Run(ctx)

	// WebRTC handler (track peer count for status)
	webrtcHandler := stream.NewWebRTCHandler(broadcaster)

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
		json.NewEncoder(w).Encode(map[string]any{
			"genre":            djStatus.CurrentGenre,
			"auto_dj":          djStatus.AutoDJ,
			"dwell_remaining":  djStatus.DwellRemaining,
			"queue_size":       djStatus.QueueSize,
			"track_id":         track.ID,
			"position":         pos.Seconds(),
			"duration":         dur.Seconds(),
			"http_listeners":   broadcaster.ListenerCount(),
			"webrtc_listeners": webrtcHandler.PeerCount(),
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

	log.Printf("drift radio live on %s", addr)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}
