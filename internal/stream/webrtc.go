package stream

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/satindergrewal/drift/internal/audio"
	"gopkg.in/hraban/opus.v2"
)

// WebRTCHandler serves WebRTC SDP negotiation for low-latency Opus streaming.
type WebRTCHandler struct {
	broadcaster *Broadcaster
	mu          sync.Mutex
	peers       []*webrtc.PeerConnection
}

// NewWebRTCHandler creates a WebRTC stream handler.
func NewWebRTCHandler(b *Broadcaster) *WebRTCHandler {
	return &WebRTCHandler{
		broadcaster: b,
	}
}

// PeerCount returns the number of active WebRTC peers.
func (h *WebRTCHandler) PeerCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.peers)
}

func (h *WebRTCHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var offer webrtc.SessionDescription
	if err := json.NewDecoder(r.Body).Decode(&offer); err != nil {
		http.Error(w, "invalid SDP offer", http.StatusBadRequest)
		return
	}

	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		http.Error(w, "create peer connection failed", http.StatusInternalServerError)
		return
	}

	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"drift-radio",
	)
	if err != nil {
		pc.Close()
		http.Error(w, "create audio track failed", http.StatusInternalServerError)
		return
	}

	if _, err := pc.AddTrack(audioTrack); err != nil {
		pc.Close()
		http.Error(w, "add track failed", http.StatusInternalServerError)
		return
	}

	if err := pc.SetRemoteDescription(offer); err != nil {
		pc.Close()
		http.Error(w, "set remote description failed", http.StatusBadRequest)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		pc.Close()
		http.Error(w, "create answer failed", http.StatusInternalServerError)
		return
	}

	if err := pc.SetLocalDescription(answer); err != nil {
		pc.Close()
		http.Error(w, "set local description failed", http.StatusInternalServerError)
		return
	}

	// Wait for ICE gathering to complete
	gatherComplete := webrtc.GatheringCompletePromise(pc)
	<-gatherComplete

	h.mu.Lock()
	h.peers = append(h.peers, pc)
	h.mu.Unlock()

	log.Printf("WebRTC peer connected (total: %d)", h.PeerCount())

	// Stream audio in background
	go h.streamToPeer(pc, audioTrack)

	// Clean up on disconnect
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		if s == webrtc.PeerConnectionStateFailed ||
			s == webrtc.PeerConnectionStateClosed ||
			s == webrtc.PeerConnectionStateDisconnected {
			h.removePeer(pc)
			pc.Close()
			log.Printf("WebRTC peer disconnected (remaining: %d)", h.PeerCount())
		}
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(pc.LocalDescription())
}

func (h *WebRTCHandler) streamToPeer(pc *webrtc.PeerConnection, track *webrtc.TrackLocalStaticSample) {
	listener := h.broadcaster.Subscribe()
	defer h.broadcaster.Unsubscribe(listener)

	enc, err := opus.NewEncoder(audio.SampleRate, audio.Channels, opus.AppAudio)
	if err != nil {
		log.Printf("WebRTC: opus encoder error: %v", err)
		return
	}
	enc.SetBitrate(128000)

	opusBuf := make([]byte, 4000)

	for {
		select {
		case <-listener.done:
			return
		case frame, ok := <-listener.C:
			if !ok {
				return
			}
			n, err := enc.Encode(frame, opusBuf)
			if err != nil {
				log.Printf("WebRTC: opus encode error: %v", err)
				continue
			}
			if err := track.WriteSample(media.Sample{
				Data:     opusBuf[:n],
				Duration: audio.FrameDuration,
			}); err != nil {
				return
			}
		}
	}
}

func (h *WebRTCHandler) removePeer(pc *webrtc.PeerConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, p := range h.peers {
		if p == pc {
			h.peers = append(h.peers[:i], h.peers[i+1:]...)
			return
		}
	}
}
