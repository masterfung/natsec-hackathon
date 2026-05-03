# Security Policy

This is a hackathon prototype, not a production system. Do not deploy to handle real classified or operational data.

## Reporting a Vulnerability

If you find a security issue in this code, please open a GitHub issue marked `security` or email the maintainer.

## Secrets

- Never commit `.env`. The `.gitignore` excludes it.
- Foundry tokens should be created with a 3-day TTL for the hackathon and revoked afterward.
- Mighty Citadel API keys must be kept out of client-side JavaScript — all Citadel calls go through the Go gateway, which holds the key server-side.
- The server Ed25519 key and voiceprint salt are stored on disk in `.secrets/` for the prototype. Production should move them to KMS/HSM-backed storage.
- Identity envelopes contain an Ed25519 private key and are returned once at enrollment. Losing one means revoke and re-enroll.
- The WebSocket OOB prototype checks that the claimed official is enrolled and active, then requires the phone client to sign a server challenge with the identity envelope's Ed25519 private key before prompts are delivered.

## Threat model assumed by the demo

The demo simulates an adversary who can:

1. Inject audio (cloned voice, ultrasonic carriers, spoken prompt injection) into the operator-AI channel.
2. Upload images / PDFs containing visual prompt injections or steganographic payloads.
3. Send text messages with embedded jailbreaks or exfiltration instructions.

The defender (Mighty Morphing + Citadel) must detect these in under one second and block them before they reach the AIP Logic agent.

## Current Prototype Boundaries

- Voice embeddings are kept in the local JSON registry so the demo can run offline. Public registry endpoints expose only the SHA-512 voiceprint hash and public key.
- The signed-media protocol signs the uploaded byte stream. The future canonical 16 kHz PCM transcode step is not implemented yet, so lossy re-encoding after signing correctly fails verification.
- Phase 0 detector validation is blocked until `fixtures/user/real_01.*`, `real_02.*`, and `real_03.*` exist.
