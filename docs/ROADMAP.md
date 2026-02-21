# Roadmap

Public roadmap for Infinara. Phase 1 is the current focus.

## Phase 1: Core Radio (Current)

Self-hosted AI radio on your LAN. The foundation.

- [x] ACE-Step v1.5 integration (submit/poll/read)
- [x] Go audio pipeline (FFmpeg decode, smoothstep crossfade, 20ms frames)
- [x] Auto-DJ with 14-genre mood graph
- [x] HTTP chunked MP3 streaming (VLC, mpv, browser)
- [x] WebRTC Opus streaming (low-latency browser playback)
- [x] Dark mode web UI with genre controls
- [x] REST API (status, genre override, skip, rate)
- [x] Docker Compose (GPU + CPU containers, shared volume)
- [ ] End-to-end test on target hardware
- [ ] First release

## Phase 2: Polish + Learning

Make it smarter. Rate tracks and it learns your taste.

- [ ] Preference learning from thumbs up/down ratings
- [ ] Polished web UI (visualizations, smoother transitions)
- [ ] Mood presets (Focus, Chill, Energetic, Late Night)
- [ ] Track history and favorites
- [ ] Improved genre captions based on rating data

## Phase 3: Video + Streaming

Combine with looping video and stream to the world.

- [ ] FFmpeg video muxing (looping LoFi video + audio)
- [ ] RTMP output to YouTube Live
- [ ] HLS for mobile device support
- [ ] "Now Playing" overlay on video stream
- [ ] Chat/request integration for live streams

## Phase 4: Native Apps

Play from any device, control with your voice.

- [ ] Desktop app (Go-native or Electron)
- [ ] Mobile app (iOS/Android)
- [ ] Voice control integration
- [ ] Background audio playback
- [ ] Skip/Previous/Next from any device

## Phase 5: Dynamic Content

Let the outside world influence the mood.

- [ ] Dynamic mood graph from trending data
- [ ] Playlist import (analyze genre distribution, auto-configure)
- [ ] Custom generation workflows
- [ ] Community-shared configurations

## Future

More phases planned. Details will emerge as earlier phases ship and real usage patterns become clear.

---

*This roadmap shows the direction, not a promise. Phases may shift based on what we learn from actually using the system. Phase 1 ships first. Everything else follows.*
