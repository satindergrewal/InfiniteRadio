# Testing

How to verify drift works at each layer. Follow this order -- each step builds on the previous.

## Prerequisites

- NVIDIA GPU with 16GB+ VRAM
- Docker with [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)
- For local development: Go 1.25+, FFmpeg, libopus-dev

## 1. ACE-Step Health Check

Verify the music generation API is running and responsive.

```bash
# Start just the ACE-Step container
docker compose up acestep -d

# Wait for model loading (~2-3 minutes on first run)
# Then check health
curl http://localhost:8001/health
```

**Expected:** `{"status": "ok"}` (or similar success response)

**If it fails:**
- Check GPU is visible: `docker exec <container> nvidia-smi`
- Check VRAM: other GPU processes may be consuming memory
- Check logs: `docker compose logs acestep`

## 2. Track Generation

Verify ACE-Step can generate a complete audio file.

```bash
# Submit a generation task
curl -X POST http://localhost:8001/release_task \
  -H "Content-Type: application/json" \
  -d '{
    "caption": "Lofi hip hop beat with vinyl crackle and mellow piano",
    "audio_duration": 30,
    "inference_steps": 8,
    "seed": 42,
    "batch_size": 1,
    "audio_format": "mp3"
  }'
```

**Expected:** JSON with `task_id`

```bash
# Poll for completion (replace TASK_ID)
curl -X POST http://localhost:8001/query_result \
  -H "Content-Type: application/json" \
  -d '{"task_id_list": ["TASK_ID"]}'
```

**Expected:** Status 1 (success) with file path in result

**Verify the file:**
```bash
# Check the shared volume
docker exec <acestep_container> ls -la /app/outputs/
```

## 3. Audio Decode

Verify FFmpeg can decode ACE-Step's output to PCM.

```bash
# Copy a generated MP3 out of the container
docker cp <acestep_container>:/app/outputs/<path_to_mp3> /tmp/test.mp3

# Decode to raw PCM
ffmpeg -i /tmp/test.mp3 -f s16le -acodec pcm_s16le -ar 48000 -ac 2 /tmp/test.pcm

# Check the output
ls -la /tmp/test.pcm
# A 30-second track should be ~5.7MB (30 * 48000 * 2 * 2 bytes)
```

**Expected:** PCM file with correct size. No FFmpeg errors.

## 4. Local Build + Smoke Test

Verify the Go binary compiles and starts.

```bash
# Build locally
go build -o radio ./cmd/radio

# Start with a mock ACE-Step (or point at running container)
ACESTEP_API_URL=http://localhost:8001 \
ACESTEP_OUTPUT_DIR=/tmp/acestep-test \
RADIO_PORT=8080 \
./radio
```

**Expected:** Logs showing:
```
drift radio starting up...
Waiting for ACE-Step API to be ready...
ACE-Step API is healthy
Auto-DJ started with genre: lofi hip hop
Generating lofi hip hop track...
```

## 5. HTTP Stream Test

Verify MP3 streaming works end-to-end.

```bash
# With the radio binary running and ACE-Step generating:

# Test with curl (should receive MP3 data)
curl -s http://localhost:8080/stream | head -c 1024 | xxd | head

# Test with VLC
vlc http://localhost:8080/stream

# Test with mpv
mpv http://localhost:8080/stream
```

**Expected:** Audio plays. No buffering gaps longer than the generation time of the first track.

## 6. Web UI Test

Verify the browser interface works.

1. Open `http://localhost:8080` in a browser
2. Click the play button -- audio should start
3. Verify the genre label updates
4. Verify the progress bar moves
5. Click a genre button -- logs should show genre change
6. Click Skip -- current track should stop, next track starts
7. Toggle Auto-DJ off/on -- dwell timer should reset

## 7. API Test

Verify all REST endpoints respond correctly.

```bash
# Status
curl http://localhost:8080/api/status | jq .

# Genre change
curl -X POST http://localhost:8080/api/genre \
  -H "Content-Type: application/json" \
  -d '{"genre": "jazz"}'

# Skip
curl -X POST http://localhost:8080/api/skip

# Auto-DJ toggle
curl -X POST http://localhost:8080/api/autodj \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# Rate
curl -X POST http://localhost:8080/api/rate \
  -H "Content-Type: application/json" \
  -d '{"rating": 1}'
```

## 8. Crossfade Test

Verify smooth transitions between tracks.

1. Set a short track duration for testing: `RADIO_TRACK_DURATION=30`
2. Set crossfade to 8 seconds: `RADIO_CROSSFADE_DURATION=8`
3. Listen through at least 3 track transitions
4. Verify no silence gaps between tracks
5. Verify the blend sounds smooth (not a hard cut, not a volume bump)

Check logs for crossfade events:
```
Now playing: <task_id_1> (genre: lofi hip hop)
Crossfaded into: <task_id_2> (genre: lofi hip hop)
```

## 9. Auto-DJ Transition Test

Verify genre transitions work.

1. Set short dwell time: `RADIO_DWELL_MIN=30 RADIO_DWELL_MAX=60`
2. Watch logs for transition events:
   ```
   Auto-DJ transition: lofi hip hop -> jazz
   Generating jazz track...
   ```
3. Verify the web UI updates the current genre
4. Verify new tracks match the new genre

## 10. Full Docker Compose Test

The final integration test.

```bash
docker compose up --build
```

**Watch for this sequence in logs:**
1. ACE-Step starts, loads model, health check passes
2. Radio starts, connects to ACE-Step
3. Auto-DJ begins generating tracks
4. "drift radio live on :8080"

**Then verify:**
- Open browser on another machine: `http://<server-ip>:8080`
- Click play, hear music
- Wait for genre transition (check logs)
- Click genre buttons, verify override works
- Open VLC on another device pointing at `/stream`
- Both devices should hear the same audio (within a few seconds sync)

## Troubleshooting

| Symptom | Check |
|---------|-------|
| ACE-Step health fails | `nvidia-smi` -- GPU visible? VRAM free? Other containers hogging GPU? |
| Generation returns status=2 | ACE-Step logs: `docker compose logs acestep`. Model may not have loaded fully. |
| No audio from /stream | Is the pipeline running? Check for "Now playing" in radio logs. FFmpeg installed in container? |
| Choppy audio | Network bandwidth (MP3 is ~192kbps, should be fine on LAN). Or CPU overloaded with too many FFmpeg encoders. |
| WebRTC fails | Browser console for errors. ICE candidates may not resolve across networks. Works best on same LAN. |
| Genre doesn't change | Is Auto-DJ enabled? Check `/api/status` for `auto_dj: true`. Dwell time may not have expired yet. |
