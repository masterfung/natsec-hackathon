# Mighty Morphing Plan

This repo implements the hackathon plan in two layers.

## Phase 0: Detector Validation

Status: prepared, blocked on user voice recordings.

- `scripts/clone_voice.py` generates Cartesia and ElevenLabs clone fixtures when provider keys are set.
- `scripts/validate_detector.py` runs real and cloned fixtures through voicebio and Citadel, then writes `docs/detector_validation_results.csv` and `docs/DETECTOR_VALIDATION.md`.
- `fixtures/user/.gitignore` prevents private voice samples from being committed.

Required local files:

- `fixtures/user/real_01.wav`
- `fixtures/user/real_02.wav`
- `fixtures/user/real_03.wav`

## Phase 1: Open Registry and Signing Protocol

Implemented:

- `POST /registry/enroll` creates an active identity profile, server-side voice embedding, SHA-512 voiceprint hash, and one-time Ed25519 identity envelope.
- `GET /registry/officials` and `GET /registry/officials/{id}` expose public registry fields only.
- `POST /sign` signs media with the user's identity envelope.
- `POST /verify` verifies detached signatures against the public registry and checks speaker match when voicebio is available.
- `GET /audit` returns a server-signed hash-chained audit log.

## Phase 2: Mighty Morphing Gateway

Implemented:

- `POST /api/verify` runs passive speaker/deepfake checks and requires an OOB approval before returning `VERIFIED`.
- `GET /ws/client?official_id=...` connects the phone PWA for approval prompts after identity-envelope challenge-response.
- `/phone/{id}` in the Vite app receives prompts and sends APPROVE/DENY responses.

Prototype limitation: OOB challenge-response uses the downloaded identity envelope directly in the browser. Production should store the key in platform secure storage or WebAuthn/FIDO2 hardware.

## Verification

Run:

```bash
make test
```

This runs backend Go tests, web TypeScript checks, production web build, and Python script syntax checks.
