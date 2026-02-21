# drift

Self-hosted AI radio station that generates infinite music with smooth genre transitions. Runs on your GPU, streams to your browser.

## What It Does

- Generates AI music continuously using [ACE-Step v1.5](https://github.com/ace-step/ACE-Step-1.5)
- Auto-DJ walks a 14-genre mood graph -- ambient drifts into chillwave, chillwave into lofi, lofi into jazz
- Smoothstep crossfades between tracks (no silence, no jarring cuts)
- Stream to any device on your LAN -- browser, VLC, mpv
- Dark mode web UI with genre controls, skip, and ratings
- HTTP chunked MP3 stream (universal) + WebRTC Opus (low-latency)

## Quick Start

```bash
git clone https://github.com/satindergrewal/drift.git
cd drift
docker compose up --build
```

Open `http://<your-server-ip>:8080` in your browser. Hit play.

Or stream directly in VLC:
```bash
vlc http://<your-server-ip>:8080/stream
```

## Requirements

- NVIDIA GPU with 16GB+ VRAM (24GB+ recommended)
- Docker with [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)
- That's it. Everything else runs in containers.

## Architecture

![Architecture](docs/images/architecture.svg)

**Audio pipeline:**

![Audio Pipeline](docs/images/audio-pipeline.svg)

**Streaming:**

![Streaming](docs/images/streaming.svg)

## Configuration

All settings via environment variables in `docker-compose.yml`:

| Variable | Default | Description |
|----------|---------|-------------|
| `ACESTEP_API_URL` | `http://acestep:8000` | ACE-Step API endpoint |
| `ACESTEP_OUTPUT_DIR` | `/acestep-outputs` | Shared volume mount point |
| `RADIO_PORT` | `8080` | HTTP server port |
| `RADIO_GENRE` | `lofi hip hop` | Starting genre |
| `RADIO_TRACK_DURATION` | `180` | Track length in seconds |
| `RADIO_CROSSFADE_DURATION` | `8` | Crossfade length in seconds |
| `RADIO_BUFFER_AHEAD` | `3` | Tracks to pre-generate |
| `RADIO_DWELL_MIN` | `300` | Min seconds per genre (Auto-DJ) |
| `RADIO_DWELL_MAX` | `900` | Max seconds per genre (Auto-DJ) |
| `RADIO_INFERENCE_STEPS` | `8` | ACE-Step turbo steps (lower = faster) |

## Genres

The Auto-DJ walks a mood graph. Transitions only follow edges -- no jumping across the map.

![Mood Graph](docs/images/mood-graph.svg)

Override via the web UI or API: `POST /api/genre {"genre": "jazz"}`

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/` | GET | Web UI |
| `/stream` | GET | Chunked HTTP MP3 stream |
| `/offer` | POST | WebRTC SDP offer/answer |
| `/api/status` | GET | Current genre, track position, queue size, listener count |
| `/api/genre` | POST | Set genre `{"genre": "jazz"}` |
| `/api/skip` | POST | Skip current track |
| `/api/autodj` | POST | Toggle Auto-DJ `{"enabled": true}` |
| `/api/rate` | POST | Rate track `{"rating": 1}` (1 = thumbs up, -1 = thumbs down) |

## Project Structure

```
drift/
+-- cmd/radio/main.go          # Entrypoint
+-- internal/
|   +-- config/config.go       # Environment-based configuration
|   +-- acestep/client.go      # ACE-Step API client
|   +-- audio/
|   |   +-- audio.go           # Constants (48kHz, 20ms frames)
|   |   +-- decoder.go         # FFmpeg subprocess: MP3 -> PCM
|   |   +-- crossfade.go       # Smoothstep crossfade
|   |   +-- pipeline.go        # Master clock, decode, mix, output
|   +-- autodj/
|   |   +-- graph.go           # 14-genre mood graph
|   |   +-- prompts.go         # Genre -> ACE-Step caption mapping
|   |   +-- scheduler.go       # Genre timing, track generation
|   +-- stream/
|   |   +-- broadcaster.go     # Fan-out: one source -> N listeners
|   |   +-- http.go            # Chunked HTTP MP3 stream
|   |   +-- webrtc.go          # Pion WebRTC + Opus
|   +-- web/
|       +-- ui.go              # go:embed for HTML
|       +-- index.html         # Dark mode web UI
+-- Dockerfile                 # Multi-stage: Go build + FFmpeg runtime
+-- docker-compose.yml         # Two services + shared volume
```

## Development

Build locally (requires Go 1.25+, FFmpeg, libopus):

```bash
go build -o radio ./cmd/radio
```

Run without Docker (point at a running ACE-Step instance):

```bash
ACESTEP_API_URL=http://localhost:8000 ./radio
```

## Acknowledgments

This project is inspired by and built upon [InfiniteRadio](https://github.com/LaurieWired/InfiniteRadio) by [LaurieWired](https://twitter.com/lauriewired). Her original project pioneered the concept of context-aware infinite music generation. The WebRTC streaming architecture, Opus encoding configuration, and crossfade algorithms in this project are adapted from her work.

LaurieWired is an exceptional engineer -- check out her work.

## License

Apache License 2.0 -- see [LICENSE](LICENSE) for details.
