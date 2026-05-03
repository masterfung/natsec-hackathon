# Mighty Morphing — Voice Biometric Sidecar

Python FastAPI service that extracts and matches speaker embeddings using
[SpeechBrain ECAPA-TDNN](https://huggingface.co/speechbrain/spkrec-ecapa-voxceleb).

The Go gateway calls this for `/api/enroll` (extract embedding) and
`/api/auth` (extract + match against `MM_Official.voice_embedding`).

## Quickstart

```bash
cd voicebio
python3.12 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# required for browser-recorded WebM/Opus or MP4 samples
ffmpeg -version

# pre-warm model on first run (~80MB download, ~10s)
uvicorn main:app --host 0.0.0.0 --port 7100
```

Health check:
```bash
curl http://localhost:7100/health
# {"status":"ok","model_loaded":true,"embed_dim":192,"sample_rate":16000,...}
```

## Endpoints

### `POST /embed`

Extract a 192-dim ECAPA-TDNN embedding from an audio file. WAV/FLAC/OGG/AIFF
decode through libsndfile; browser-recorded WebM/Opus and MP4 decode through
`ffmpeg`. All inputs are auto-resampled to 16kHz mono.

```bash
curl -F "audio=@vance_30s.wav" http://localhost:7100/embed
```

Response:
```json
{
  "embedding": [0.034, -0.119, ...],
  "dim": 192,
  "duration_s": 31.4,
  "quality": 0.82,
  "inference_ms": 89
}
```

### `POST /match`

Cosine similarity between two embeddings.

```bash
curl -X POST http://localhost:7100/match \
  -H "Content-Type: application/json" \
  -d '{"enrolled":[...],"candidate":[...]}'
```

Response:
```json
{"similarity": 0.94, "threshold": 0.75, "is_match": true}
```

### `GET /health`

Liveness + model load status.

## Configuration

| env | default | purpose |
|---|---|---|
| `MM_SPEAKER_MATCH_THRESHOLD` | 0.75 | default match threshold (overridable per request) |
| `VOICEBIO_MODEL_CACHE` | `/tmp/speechbrain-cache` | local cache dir for the ECAPA model |

## Notes on accuracy

ECAPA-TDNN trained on VoxCeleb is the strongest open speaker-recognition model
that runs on CPU at sub-100ms latency. EER on VoxCeleb1-cleaned is ~0.69%.

For the hackathon demo, **0.75 cosine threshold** has been shown to give clean
separation between same-speaker (typically 0.85+) and different-speaker
(typically <0.5) on press-conference style audio at 30s+ enrollment length.

## Performance

| Op | Cold start | Warm |
|---|---|---|
| Model load | ~10s (first run) | n/a |
| `/embed` (30s audio) | ~250ms | ~80ms |
| `/match` | <2ms | <2ms |
