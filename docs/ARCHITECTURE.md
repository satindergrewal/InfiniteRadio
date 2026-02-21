# Architecture

Technical design document for Infinara. Explains how the system works and why decisions were made.

## System Overview

![System Architecture](images/architecture.svg)

Infinara is two Docker containers talking over an internal network with a shared volume. The **acestep** container runs ACE-Step v1.5 on the GPU for music generation. The **radio** container is a Go binary handling everything else -- streaming, Auto-DJ, web UI, API.

## Audio Pipeline

![Audio Pipeline](images/audio-pipeline.svg)

### Frame Size

All audio processing uses 20ms frames at 48kHz stereo int16:
- 960 samples per channel per frame
- 1920 total interleaved samples
- 3840 bytes per frame

20ms was chosen because it's the standard Opus frame size. Matching it avoids resampling in the WebRTC path.

### Decode

FFmpeg runs as a subprocess per track. The entire MP3 is decoded to PCM in memory. A 3-minute track at 48kHz stereo int16 is ~33MB. With 3 tracks buffered, ~100MB -- well within tolerance.

Full decode (vs streaming decode) is required because crossfading needs access to frames near the end of the outgoing track and the beginning of the incoming track simultaneously.

### Crossfade

Smoothstep curve: `3t^2 - 2t^3` where t goes from 0 to 1 across the crossfade duration.

Same curve as the original InfiniteRadio Python code. Produces a natural-sounding blend -- slow start, fast middle, slow end. Linear crossfade sounds abrupt at the boundaries.

The pipeline pre-decodes the next track in a background goroutine (capacity: 4). When the current track enters the crossfade zone, frames from both tracks are blended. After the crossfade, playback continues from where the incoming track left off.

### Master Clock

The pipeline outputs frames at real-time rate using `time.Ticker` at 20ms intervals. This is the master clock for the entire system. Without pacing, FFmpeg would encode everything instantly and listeners would get a burst of audio followed by silence.

## Streaming

![Streaming Architecture](images/streaming.svg)

### Broadcaster

Fan-out pattern: one PCM source to N listeners. Each listener gets a buffered channel (~3 seconds of frames). Slow listeners get frames dropped rather than blocking the broadcast. One slow client never stalls everyone else.

### HTTP (MP3)

Each HTTP connection spawns its own FFmpeg process: `PCM frames -> FFmpeg stdin -> MP3 bytes -> HTTP response (chunked)`.

Per-connection FFmpeg means each listener gets a valid MP3 stream from the moment they connect. Shared encoding would require all listeners to join at a keyframe boundary.

Trade-off: more CPU for multiple listeners. At the target scale (1-5 LAN listeners), this is fine. For 100+ listeners, a shared encoder with ring buffer would be needed.

### WebRTC (Opus)

Each WebRTC peer gets an Opus encoder (128kbps, 48kHz, stereo). Opus is encoded in Go via `gopkg.in/hraban/opus.v2` (CGo binding to libopus). Frames are sent as RTP packets via Pion WebRTC v4.

WebRTC offers lower latency (~50ms vs ~2-5 seconds for HTTP), but requires browser JavaScript for SDP negotiation.

## Auto-DJ

![Mood Graph](images/mood-graph.svg)

### Mood Graph

14 genres connected by affinity edges. The DJ can only transition to adjacent genres -- no jumping from ambient to drum and bass.

The graph topology was designed so that:
- Calm genres cluster together (ambient, chillwave, classical)
- Energetic genres cluster together (electronic, DnB, rock)
- Bridge genres connect the clusters (synthwave, indie rock)
- Every genre is reachable from every other genre (connected graph)

### Scheduling

The scheduler loop:
1. Check if generation buffer is below threshold (default: 3 tracks)
2. If so, submit a generation job to ACE-Step and poll until complete
3. Push completed track to the pipeline queue
4. Check if dwell time expired (5-15 minutes per genre, randomized)
5. If expired and Auto-DJ is on, pick a random adjacent genre

Manual genre override resets the dwell timer immediately. Tracks already in the queue play out normally.

### Genre Captions

Each genre maps to a 15-25 word caption sent to ACE-Step describing instruments, mood, tempo, and production style. All instrumental in Phase 1.

## ACE-Step Integration

ACE-Step v1.5 exposes a REST API:

1. `POST /release_task` -- submit generation with caption, duration, format, seed, inference steps
2. `POST /query_result` -- poll with task ID, returns status (0=running, 1=success, 2=failed)
3. On success, result contains a file reference

The client checks the shared volume first (direct file read, zero network overhead). Falls back to HTTP download if file isn't found on the shared volume.

Turbo mode (8 inference steps) generates a 3-minute track in ~5-8 seconds on RTX 4090 class hardware.

## Why Two Containers

| Concern | Single container | Two containers (chosen) |
|---------|-----------------|------------------------|
| GPU isolation | Shared VRAM between Go and Python | GPU exclusively for generation |
| Updates | Rebuild everything for any change | Update radio or ACE-Step independently |
| Debugging | Interleaved logs, shared process space | Clean separation, independent logs |
| Audio transfer | In-process, fastest | Shared volume (disk read), negligible overhead |

The shared volume avoids the ~7MB HTTP download per track that would otherwise be needed.

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Go over Python | Single binary, excellent concurrency, FFmpeg/Opus via subprocess/CGo |
| FFmpeg subprocess over Go audio libs | FFmpeg handles every codec. Go audio libraries are fragmented. |
| Shared volume over HTTP download | ~7MB per track. Disk read is instant vs network overhead. |
| Per-listener FFmpeg over shared encoder | Simpler. Valid stream from connection start. Fine for LAN scale. |
| 20ms frames | Matches Opus standard. No resampling needed in WebRTC path. |
| Smoothstep over linear crossfade | Natural blend. Proven in original InfiniteRadio. |
| Embedded HTML over separate frontend | Zero build tooling. Single binary deployment. |
| Channel-based broadcaster | Go-idiomatic. Backpressure via capacity. Drop semantics for slow listeners. |
| Background decoder goroutine | Pre-decodes next track while current plays. No decode stall during crossfade. |
