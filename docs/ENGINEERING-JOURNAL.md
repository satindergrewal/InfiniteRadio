# Engineering Journal

Architecture Decision Records (ADRs) and technical decisions for drift. Documents WHY, not just WHAT.

---

## ADR-001: Go Over Python for the Radio Service

**Date:** 2025-02-21
**Status:** Accepted

**Context:** The original InfiniteRadio uses Python for everything (music generation, streaming, DJ logic). We're replacing the music generation model (Magenta RT -> ACE-Step v1.5), which already has its own Python REST API.

**Decision:** Write the radio service in Go. Keep ACE-Step in Python (it has to be -- PyTorch).

**Rationale:**
- Single binary deployment. No virtualenvs, no pip, no dependency hell in production.
- Go's concurrency model (goroutines, channels) maps perfectly to the broadcaster fan-out pattern.
- Real-time audio requires predictable latency. Go's GC pauses are sub-millisecond. Python's GIL makes concurrent audio processing painful.
- ACE-Step already exposes a REST API. The boundary between Go and Python is clean HTTP.

**Trade-offs:**
- CGo required for Opus encoding (gopkg.in/hraban/opus.v2). Adds build complexity.
- No Go equivalent for PyTorch. ML inference stays in Python forever. That's fine -- it's behind the API boundary.

---

## ADR-002: Shared Docker Volume Over HTTP Download

**Date:** 2025-02-21
**Status:** Accepted

**Context:** ACE-Step generates MP3 files. The radio service needs to read them. Two options: (a) download via HTTP from ACE-Step's API, or (b) read directly from a shared Docker volume.

**Decision:** Shared volume, with HTTP download as fallback.

**Rationale:**
- A 3-minute MP3 at 48kHz stereo is ~7MB. Reading from disk: <1ms. Downloading over HTTP (even localhost): 50-200ms + memory allocation.
- With 3 tracks buffered and continuous generation, eliminating 7MB transfers per track removes a meaningful source of latency.
- The fallback ensures the system works even if the volume mount is misconfigured.

---

## ADR-003: Per-Listener FFmpeg Over Shared Encoder

**Date:** 2025-02-21
**Status:** Accepted

**Context:** HTTP MP3 streaming requires encoding PCM to MP3. Options: (a) one FFmpeg process shared across all listeners, or (b) one FFmpeg process per listener.

**Decision:** Per-listener FFmpeg.

**Rationale:**
- Each listener gets a valid MP3 stream from the first byte. No need to sync with keyframe boundaries.
- Listeners can connect and disconnect at any time without affecting others.
- At LAN scale (1-5 listeners), the CPU overhead of multiple FFmpeg processes is negligible.

**When to revisit:** If we ever need 100+ concurrent HTTP listeners, a shared encoder with a ring buffer and keyframe injection would be more efficient. Not needed for Phase 1.

---

## ADR-004: 20ms Frame Size

**Date:** 2025-02-21
**Status:** Accepted

**Context:** Need to choose a standard frame size for the internal PCM pipeline.

**Decision:** 20ms at 48kHz = 960 samples per channel per frame.

**Rationale:**
- 20ms is the standard Opus frame duration. Using it avoids resampling or rebuffering in the WebRTC path.
- Small enough for responsive skip/crossfade behavior (~50 frames per second).
- Large enough that per-frame overhead (channel sends, goroutine scheduling) is negligible.

---

## ADR-005: Smoothstep Crossfade

**Date:** 2025-02-21
**Status:** Accepted

**Context:** Need a crossfade curve between tracks.

**Decision:** Smoothstep: `3t^2 - 2t^3`, same as the original InfiniteRadio.

**Rationale:**
- Proven to sound good. The original project validated this curve.
- Slow start and slow end prevent the "volume bump" artifact that linear crossfade produces at the boundaries.
- Mathematically simple. No lookup tables, no approximations.

**Alternatives considered:**
- Linear (`t`): too abrupt at boundaries
- Cosine (`(1 - cos(pi*t))/2`): similar result but more computation for no perceptible benefit
- Equal-power (sqrt-based): standard in DAWs but not necessary for genre transitions where precise energy preservation isn't critical

---

## ADR-006: Background Decoder Goroutine

**Date:** 2025-02-21
**Status:** Accepted

**Context:** FFmpeg decoding a 3-minute MP3 takes ~0.5-1 second. If decoding happens inline during the crossfade zone, there's a gap in audio output.

**Decision:** Separate goroutine decodes tracks as they arrive, feeding decoded PCM into a buffered channel (capacity 4).

**Rationale:**
- Decoding is always ahead of playback. By the time the pipeline needs the next track, it's already decoded.
- The buffered channel provides natural backpressure -- if 4 tracks are decoded and waiting, the decoder blocks.
- Keeps the main pipeline goroutine focused on frame output timing (master clock).

---

## ADR-007: Channel-Based Broadcaster

**Date:** 2025-02-21
**Status:** Accepted

**Context:** Need to distribute PCM frames from one source to multiple listeners.

**Decision:** Go channels with select-based drop semantics.

**Rationale:**
- Each listener gets a buffered channel (150 frames, ~3 seconds). The broadcaster sends frames to all listeners via non-blocking select.
- If a listener's channel is full (slow consumer), the frame is dropped. This prevents one slow client from stalling the entire broadcast.
- Go channels handle the synchronization. No mutexes needed in the hot path (except the listener map, which is read-locked).

**Alternative considered:** Shared ring buffer. More memory-efficient for many listeners, but adds complexity (reader tracking, wrap-around logic). Channels are simpler and sufficient for LAN scale.

---

## ADR-008: Embedded HTML Over Separate Frontend

**Date:** 2025-02-21
**Status:** Accepted

**Context:** Need a web UI for controlling the radio.

**Decision:** Single HTML file embedded via `go:embed`. No JavaScript framework, no build step.

**Rationale:**
- Zero build tooling. No node_modules, no webpack, no npm. The Go binary IS the deployment.
- The UI is a control panel, not an application. It needs genre buttons, play/pause, status display. Vanilla JS handles this in ~150 lines.
- Polling `/api/status` every 2 seconds is simpler than WebSocket state management and sufficient for a status dashboard.

**When to revisit:** Phase 2 UI polish might warrant a lightweight framework (Preact, Svelte). But not until the current UI hits its limits.

---

## ADR-009: Original Code Preserved on Separate Branch

**Date:** 2025-02-21
**Status:** Accepted

**Context:** drift is a major rewrite of LaurieWired's InfiniteRadio. The original code no longer exists on main.

**Decision:** Created `original/lauriewired` branch to preserve the original code frozen in time.

**Rationale:**
- Honors the original author's work (Apache 2.0 requires attribution, not preservation, but respect goes further than legal requirements).
- Provides reference for anyone wanting to understand what was adapted (WebRTC patterns, Opus config, crossfade algorithm).
- Clean separation -- main branch is drift, the branch is a historical record.

---

*New ADRs are added as decisions are made. Each records the context, decision, rationale, and trade-offs. Settled decisions aren't revisited unless new information changes the context.*
