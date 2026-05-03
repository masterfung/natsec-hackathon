# Mighty Morphing — Architecture

## Mission

Internal digital defense layer for signed voice content, senior-leader communications, and AI agent input/output integrity, built around an open registry/signing protocol plus Mighty Citadel's multimodal AI security stack.

## Layers

Layer 1 is standalone and does not require Mighty Morphing:

- `POST /registry/enroll` publishes a profile and returns a one-time Ed25519 identity envelope.
- `GET /registry/officials/{id}` returns public registry fields only: ID, name, role, SHA-512 voiceprint hash, public key, status, enrollment center, and timestamps.
- `POST /sign` creates a detached media signature envelope.
- `POST /verify` verifies the detached signature against the registry and checks speaker match when voicebio is available.
- `GET /audit` exposes the hash-chained, server-signed audit log.

Layer 2 is the premium gateway:

- `POST /api/verify` runs speaker match + Citadel deepfake detection + phone OOB approval.
- `GET /ws/client?official_id=...` connects the phone PWA for OOB prompts.
- `GET /api/events` streams SOC events over SSE.

## High-level diagram

```
┌──────────────────────────────────────────────────────────────────────────────┐
│  SOC Console (Foundry Workshop primary; Vite fallback at /natsec/web)        │
│                                                                                │
│  ┌─────────────────┐ ┌──────────────────────┐ ┌──────────────────────────┐ │
│  │  Enrolled       │ │  Live Comms Feed     │ │  Incident Timeline       │ │
│  │  Officials      │ │   ─ voice            │ │   + AIP Synthesis Brief  │ │
│  │  Roster         │ │   ─ text             │ │                          │ │
│  │  + Enroll btn   │ │   ─ document         │ │  per incident:           │ │
│  │                 │ │   ─ image            │ │   ─ MM_AuthEvent refs    │ │
│  │  per row:       │ │  per event:          │ │   ─ AIP Logic 2-3 line   │ │
│  │  ─ name + photo │ │   ─ verdict          │ │     synthesis brief      │ │
│  │  ─ enroll date  │ │   ─ speaker_match    │ │   ─ recommended COA      │ │
│  │  ─ embedding    │ │   ─ deepfake_risk    │ │                          │ │
│  │    quality      │ │   ─ injection_risk   │ │                          │ │
│  │                 │ │   ─ latency_ms       │ │                          │ │
│  └─────────────────┘ └──────────────────────┘ └──────────────────────────┘ │
└────────────────────────────────────┬───────────────────────────────────────────┘
                                     │ HTTP + Server-Sent Events
                                     v
┌──────────────────────────────────────────────────────────────────────────────┐
│  Mighty Morphing Gateway (Go 1.26, :7000)                                    │
│                                                                                │
│   POST /registry/enroll         audio → registry profile + identity envelope │
│   POST /sign                    audio + identity → detached signature        │
│   POST /verify                  audio + signature → standalone verification  │
│   GET  /audit                   hash-chained signed audit log                │
│   POST /api/auth                audio → 2-factor → MM_AuthEvent              │
│   POST /api/verify              audio → voice + deepfake + OOB verdict       │
│   GET  /ws/client               phone PWA WebSocket                          │
│   POST /api/scan/text           text → Citadel text → MM_AuthEvent           │
│   POST /api/scan/document       PDF  → Citadel vision/text → MM_AuthEvent    │
│   POST /api/scan/image          image → Citadel vision → MM_AuthEvent        │
│   POST /api/output              outbound LLM text → exfil + secrets check    │
│   GET  /api/officials           list enrolled officials                       │
│   GET  /api/events              SSE stream for live console                  │
│   GET  /health, /api/config-check                                            │
│                                                                                │
│  Holds: CITADEL_API_KEY, FOUNDRY_TOKEN  (never sent to browser)              │
└──────┬─────────────────────┬───────────────────────────────┬──────────────────┘
       │                     │                               │
       v                     v                               v
┌──────────────────┐ ┌─────────────────────────┐ ┌────────────────────────────┐
│  Voice Biometric │ │  Mighty Citadel         │ │  Palantir Foundry / AIP    │
│  Sidecar         │ │  (deployed, hosted)     │ │  NatSec Hackathon Ontology │
│  (Python, :7100) │ │                         │ │  ri.ontology.main.ontology │
│                  │ │  - audio WS/REST        │ │  .41fccd0c-...              │
│  POST /embed     │ │    deployed v16 audio   │ │                             │
│  POST /match     │ │    99% TPR              │ │  + MM_Official              │
│  GET  /health    │ │  - vision REST          │ │  + MM_VoiceBiometric        │
│                  │ │    98.67% TPR           │ │  + MM_Communication         │
│  SpeechBrain     │ │    AEROBLADE            │ │  + MM_AuthEvent             │
│  ECAPA-TDNN      │ │  - text gateway         │ │                             │
│  192-dim float32 │ │    1200+ patterns       │ │  Reused: RouteAlert,        │
│                  │ │    mmBERT v5.7 FP8      │ │  RouteAlertComment, Drones, │
│                  │ │    XLM-R v5.7.17 FP8    │ │  Aircraft                   │
│                  │ │                         │ │                             │
│                  │ │                         │ │  AIP Logic agent:           │
│                  │ │                         │ │  IncidentSynthesizer        │
└──────────────────┘ └─────────────────────────┘ └────────────────────────────┘
```

## Three-factor gateway verification

`POST /api/verify` is the demo gateway path. It first computes the passive checks below. If they pass, it sends an OOB prompt to the phone PWA. `APPROVE` returns `VERIFIED`; `DENY`, timeout, Citadel failure, high deepfake risk, or speaker mismatch returns `BLOCKED`.

## Two-factor passive authentication

Every voice channel into a command-grade decision passes both checks. Adversary must break both simultaneously to succeed.

### Factor 1: Speaker match (identity)

1. Audio sent to voicebio sidecar `POST /embed`.
2. SpeechBrain ECAPA-TDNN model (HuggingFace `speechbrain/spkrec-ecapa-voxceleb`) extracts a 192-dim float32 embedding.
3. Cosine similarity computed vs. enrolled `MM_Official.voice_embedding`.
4. **Pass** if `similarity >= MM_SPEAKER_MATCH_THRESHOLD` (code default 0.75; `.env.example` uses 0.59 for the controlled demo fixtures, with OOB as the command-grade gate).
5. Latency: ~50-100ms on CPU after warm start.

### Factor 2: Deepfake risk (authenticity)

1. Same audio sent to Mighty Citadel `POST /v1/audio/scan` (`mode=secure`).
2. Deployed v16 Wav2Vec2/WavLM ensemble + RMT divergence + ASR-prefix-attack/text-pipeline signals.
3. Returns `risk` 0-100 with full signal breakdown.
4. **Pass** if `risk < MM_DEEPFAKE_BLOCK_THRESHOLD` (default 80).
5. Latency: ~287ms p95 measured live on Akamai.

### Combined verdict matrix

| Speaker match | Deepfake risk | Verdict | Why |
|---|---|---|---|
| ≥ 0.75 | < 80 | **VERIFIED** | Both factors pass |
| ≥ 0.75 | ≥ 80 | **BLOCKED** | High-fidelity clone of enrolled official |
| < 0.75 | < 80 | **BLOCKED** | Real-but-wrong-person (impersonator) |
| < 0.75 | ≥ 80 | **BLOCKED** | Synthetic audio of unknown speaker |

A high-quality clone can sometimes pass a single classifier, which is why the gateway does not rely on classifier-only trust. To authorize a command-grade action, the audio must match the enrolled speaker, stay below deepfake and spoken-injection block thresholds, resolve through the registry, and receive out-of-band approval from the enrolled device.

## AI agent input / output guardrail

Beyond voice, every input the agent ingests and every output it produces is scanned through Mighty Citadel.

| Vector | Citadel surface | Catches |
|---|---|---|
| PDF intel report | Vision + text gateway | OCR'd prompt injection, embedded steg, hidden instructions, document forensics |
| Uploaded image (e.g., "official photo", "satellite") | Vision | AI-generated detection (98.67% TPR), manipulation (AEROBLADE), steg, OCR injection |
| Text message / chat | Text gateway | 1,200+ injection patterns, mmBERT v5.7 + XLM-R v5.7.17 toxicity ensemble |
| Outbound AI response | Text gateway (output mode) | Secrets, classified markers, PII, exfil patterns |

Every scan produces an `MM_AuthEvent` with the same schema, regardless of modality. Cross-modal corroboration is then computed in the gateway.

## Cross-modal corroboration

Single-modality detection misses coordinated attacks. Mighty Morphing sees the whole packet.

If any 30-second window contains BLOCK-level findings on **two or more modalities** (e.g., voice deepfake AND text injection AND AI-gen image), the gateway boosts confidence and creates a high-priority `RouteAlert` with `kind=multi_vector_attack`. AIP Logic synthesizer flags this as APT-style coordinated activity.

This is the gap research confirmed nobody else fills (Reality Defender + Pindrop + Lakera together don't have this).

## Demo flow (60s)

| t | Event |
|---|---|
| 0:00–0:08 | Enroll a controlled demo official from consented audio. Console: `ENROLLED · embedding-quality 0.92 · 8.4s`. |
| 0:08–0:18 | Genuine call routes through the approval device. `VERIFIED · speaker-match pass · deepfake-risk pass · OOB approved`. |
| 0:18–0:35 | Controlled cloned voice attempts a command phrase. `BLOCKED · clone/injection signals elevated · OOB skipped or denied`. RouteAlert AT-2026-001 created. AIP Logic synthesis: *"Multi-vector spoofing pattern matches coordinated social-engineering TTPs. Hold + escalate to verification cell."* |
| 0:35–0:48 | PDF prompt injection BLOCKED in 740ms. AI-gen "official photo" risk 0.91. Outbound AI reply has 1 secret stripped. |
| 0:48–0:58 | Architecture slide with production benchmarks. |
| 0:58–1:00 | Close: *"Cabinet turns over, Mighty Morphing onboards in 60 seconds. Adversaries don't get a head start anymore."* |

## Foundry ontology

See [docs/ONTOLOGY_SETUP.md](docs/ONTOLOGY_SETUP.md) for click-by-click setup. Formal schema in [ontology/schema.md](ontology/schema.md).

### New object types (4)

| Object | Properties (PK in **bold**) |
|---|---|
| `MM_Official` | **id**, name, role, photo_uri, voice_embedding (string of 192 floats joined by `,`), enrollment_date, embedding_quality, status |
| `MM_VoiceBiometric` | **id**, official_id (FK), embedding, sample_audio_uri, quality_score, created_at |
| `MM_Communication` | **id**, channel (radio/teams/zoom/voicememo), audio_uri, transcript, sender_claimed (string), received_at |
| `MM_AuthEvent` | **id**, communication_id (FK), official_id (FK), speaker_match (double), deepfake_risk (integer), injection_risk (integer), verdict (VERIFIED/WARN/BLOCKED), latency_ms (integer), signals_json (string), processed_at |

### Reused object types (already in ontology)

- `RouteAlert` — surfaced human-readable threat with status workflow
- `RouteAlertComment` — audit trail comments under each alert
- `Drones`, `Aircraft` — fleet under defense (for the C2 narrative slide)

### New action types (2)

| Action | Operation | Purpose |
|---|---|---|
| `mm-enroll-official` | createObject | Create MM_Official from voicebio embedding output |
| `mm-log-auth-event` | createObject | Create MM_AuthEvent after every auth check |

### Reused actions

- `add-route-alert-comment...` — attach forensic detail to RouteAlert
- `[Example] Update Route Alert Status` — escalate / close

## AIP Logic agent: IncidentSynthesizer

Built in AIP Logic UI (no-code), invoked from the gateway after every BLOCKED event.

**Inputs:**
- Recent `MM_AuthEvent` objects (last 30 minutes)
- Recent `RouteAlert` objects

**Output:**
- 2-3 line synthesis brief: pattern, suspected actor profile, recommended COA
- Optionally appended as a `RouteAlertComment` to the latest alert

This is the "AIP using AIP" demo moment — a real Foundry-hosted LLM agent producing operational summaries, grounded in our own ontology data.

## Configuration

See [.env.example](.env.example). Key thresholds:

| Var | Default | Purpose |
|---|---|---|
| `MM_SPEAKER_MATCH_THRESHOLD` | 0.75 code default / 0.59 demo env | Cosine similarity floor for speaker match; OOB carries command authorization |
| `MM_DEEPFAKE_BLOCK_THRESHOLD` | 80 | Citadel deepfake risk above this → BLOCK |
| `MM_INJECTION_BLOCK_THRESHOLD` | 70 | Citadel injection risk above this → BLOCK |
| `MM_AUTH_LATENCY_BUDGET_MS` | 500 | Soft target; logged for SLA tracking |
| `VOICEBIO_URL` | http://localhost:7100 | Voice biometric sidecar URL |

## Why this maps cleanly to PS4

PS4 example #3 verbatim: *"a deployable security scanning toolkit that validates [AI deployments] against known-good baselines, detecting anomalous files, tampered libraries, or embedded threats before models influence operational decisions."*

| PS4 phrase | Mighty Morphing answer |
|---|---|
| validates against known-good baselines | enrolled voice biometric IS the known-good baseline for an official |
| anomalous files / tampered libraries / embedded threats | cloned voice clips, AI-gen images, prompt-injected PDFs, exfil-attempting LLM outputs |
| before models influence operational decisions | gateway BLOCKs at the input layer; agent never sees poisoned content |

Plus PS4's broader mission: *"protect AI deployments and communications links, harden the digital backbone."* Voice channels into AI agents ARE the communications links; multimodal scan IS the digital backbone hardening.
