# Mighty Morphing

**Internal digital defense layer for senior-leader communications and AI agent integrity.**

Built for the [3rd Annual NatSec Hackathon](https://cerebralvalley.ai/e/3rd-annual-natsec-hackathon) (May 2-3, 2026), Problem Statement 4: Digital Defense and Cybersecurity.

---

## The dire pain

Adversaries can clone convincing voice samples from short recordings. Recent public incidents show the operational shape of the threat:

- **July 2025** — An impostor used Marco Rubio's cloned voice to call foreign ministers and US officials. FBI confirmed the campaign was ongoing. ([NPR](https://www.npr.org/2025/07/10/nx-s1-5462844/state-department-investigating-incident-in-which-ai-used-to-impersonate-marco-rubio))
- **January 2024** — A fake-Biden robocall told New Hampshire voters to stay home. ([BBC](https://www.bbc.com/news/world-us-canada-68064247))
Cabinet officials and senior operational leaders change regularly. Defenders need fast onboarding, public provenance, and fail-closed verification before synthetic media reaches decision systems. **That gap is what gets exploited.**

## What Mighty Morphing does

A two-layer protection system for signed voice content and senior-leader communications:

1. **Open identity registry + signing protocol.** Enrollment creates an Ed25519 identity envelope, publishes a public key + SHA-512 voiceprint hash, and supports detached media signatures that anyone can verify against the registry.
2. **Standalone verification.** Upload audio + `.sig` and the gateway verifies signature validity, registry status, and speaker match when voicebio is available.
3. **Mighty Morphing gateway.** Runtime voice verification combines:
   - **Speaker match** — does this audio belong to the enrolled official? (cosine similarity vs. their stored embedding)
   - **Deepfake risk** — is the audio synthetic? (Mighty Citadel Wav2Vec2/WavLM ensemble, deployed v16 head)
   - **OOB confirmation** — phone PWA receives a WebSocket prompt and must APPROVE before `/api/verify` returns `VERIFIED`.
4. **Hash-chained audit trail.** Enrollment, signing, verification, and gateway decisions append to a server-signed audit log exposed at `/audit`.
5. **Minimal web console.** Pages are available for `/enroll`, `/sign`, `/verify`, `/registry`, `/audit`, `/phone/{id}`, and `/`.

## Implementation status

Implemented in this workspace:

- Layer 1 registry, enrollment, media signing, standalone verification, and audit log.
- Layer 2 `/api/verify` with passive checks plus WebSocket OOB approval.
- Phase 0 clone/validation harness.
- Backend handler tests covering enroll -> sign -> verify -> tamper rejection -> audit chain.

Current validation notes:

- `docs/DETECTOR_VALIDATION.md` and `docs/clone_defense_eval_live.json` contain controlled local fixture results against Cartesia and ElevenLabs clone artifacts.
- The fresh Cartesia red-team sample shows why the product is layered: pure audio classification is not enough for command-grade decisions. The gateway combines registry identity, speaker match, injection/deepfake scans, and out-of-band approval.
- The phone WebSocket supports enrolled/active official checks and challenge-response authentication when the client presents the enrolled key material.

## Detector benchmarks

| Detector | Performance | Latency |
|---|---|---|
| Voice deepfake (deployed v16 audio stack) | **99.14% TPR ElevenLabs · 97.73% AUDETER · 98.90% full attack**; real BLOCK FPR 0.98%, multilingual BLOCK FPR 0.00% | sub-second end-to-end |
| Multimodal voice corroboration | BLOCK risk 96 on cloned + injected | <500ms p95 |
| Spoken prompt injection (Qwen3-ASR + mmBERT) | 94.5% BLOCK | sub-second |
| Ultrasonic / DolphinAttack | BLOCK risk 79-88 | ~150ms |
| AI-generated images (general) | **98.67% TPR / 2.04% FPR** | sub-second |
| Image manipulation (AEROBLADE) | 81% TPR @ 5% FPR | sub-second |
| Text prompt injection | 1,200+ patterns + mmBERT v5.7 + XLM-R v5.7.17 | <10ms |

These are Mighty Citadel deployment benchmarks. The hackathon-specific evidence for this repo is the controlled fixture evaluation in `docs/clone_defense_eval_live.json`: Cartesia clones detected at warn-or-block on 8/12 fixtures, ElevenLabs artifacts on 9/12 fixtures, and one fresh Cartesia hard-negative that bypassed the pure classifier but is still handled by the command gateway's fail-closed OOB path.

## OSS data layer

The passive spoof detector now has a normalized public-corpus ingestion layer:

- `scripts/build_audio_manifest.py` converts local fixtures, ASVspoof, MLAAD, MultiAPI-Spoof, In-The-Wild, WaveFake, SINE, or generic metadata into one manifest schema.
- `scripts/extract_spoof_features.py --manifest ...` runs the same feature pipeline over any manifest.
- `scripts/train_spoof_meta.py --group-by dataset --group-by generator_id` reports leave-one-group-out holdouts so we can measure whether the detector generalizes beyond one clone provider.
- Large audio is pushed to Modal by default via `make modal-put-fixtures` and `MODAL_VOLUME=citadel-audio-oss-data`.

See [docs/OSS_DATA_LAYER.md](docs/OSS_DATA_LAYER.md).

## Why nobody else has this

Independent research (May 2026) confirms: **no commercial vendor or DoD program has production-grade simultaneous multimodal AI security with cross-modal corroboration at sub-500ms.**

| Player | What they do | Gap |
|---|---|---|
| Lakera, Protect AI, HiddenLayer | Text-only LLM security, <50ms | No voice, no vision, no steg |
| Reality Defender | Audio + video deepfake | No text injection, no document/image |
| Pindrop | Voice/call channel | No multimodal, no document/image |
| DARPA SemaFor | Concluded September 2024 | No longer a program |
| Palantir/Anduril Golden Dome | Sensor fusion C2 ($185B) | Doesn't gate input authenticity |

Mighty Morphing fills the gap — using Mighty Citadel as the deployed multimodal brain.

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) for full detail. High level:

```
Foundry Workshop SOC Console (or Vite fallback)
         │
         v
Go Gateway (:7000) ─┬─→ Mighty Citadel (audio/vision/text — multimodal)
                    ├─→ Voice Biometric Sidecar (Python, SpeechBrain ECAPA-TDNN)
                    └─→ Palantir Foundry Platform (ontology + AIP Logic)
```

## Stack

| Layer | Tech |
|---|---|
| SOC Console | Foundry Workshop (primary, AIP-native); Vite + React 19 + TS + Tailwind v4 (fallback) |
| Gateway | Go 1.26, stdlib `net/http`, `slog` |
| Voice biometric | Python 3.12, FastAPI, SpeechBrain ECAPA-TDNN (192-dim embeddings) |
| AI security | Mighty Citadel (deployed, hosted) |
| Ontology + Agent | Palantir Foundry / AIP Logic |

## Quickstart

```bash
# 1. Drop credentials in .env
cp .env.example .env  # then fill in CITADEL_API_KEY, FOUNDRY_TOKEN, etc.

# 2. Set up the Foundry ontology (one-time, ~10 min of UI clicks)
# Follow docs/ONTOLOGY_SETUP.md to create the 4 MM_ object types + 2 actions

# 3. Run all three services
make dev   # starts voicebio :7100, gateway :7000, web :5173 concurrently
```

Open `http://localhost:5173`.

Key pages:

| Page | Purpose |
|---|---|
| `/enroll` | Create registry profile and download identity envelope |
| `/sign` | Sign audio with an identity envelope |
| `/verify` | Standalone signature + speaker verification |
| `/registry` | Public registry view |
| `/audit` | Hash-chain audit log |
| `/phone/user-self` | OOB approval PWA |
| `/` | SOC event console |

Run verification:

```bash
make test
```

## Ports

| Service | Port |
|---|---|
| SOC console (Vite dev) | 5173 |
| Go gateway | 7000 |
| Voice biometric sidecar | 7100 |
| Mighty Citadel (external Akamai) | per env |

## Project structure

```
natsec/
├── backend/            Go gateway (single binary)
│   ├── main.go         routes + glue
│   └── internal/       citadel, foundry, voice, sse helpers
├── voicebio/           Python FastAPI sidecar (SpeechBrain ECAPA-TDNN)
├── web/                Vite SOC console fallback
├── ontology/           ontology spec (schema.md)
├── docs/               setup guides (ONTOLOGY_SETUP.md, etc.)
├── fixtures/           demo audio/PDF/image fixtures
└── scripts/            helpers
```

## License

MIT — see LICENSE.
