# Mighty Morphing — Ontology Schema (formal spec)

Reference for the four MM_ object types and two action types added to the
NatSec Hackathon Ontology.

| Field | Notes |
|---|---|
| Stack | nshackathon.palantirfoundry.com |
| Ontology RID | ri.ontology.main.ontology.41fccd0c-2180-4c1d-841d-8a488d1abb46 |
| Setup | See [docs/ONTOLOGY_SETUP.md](../docs/ONTOLOGY_SETUP.md) |

## Object types

### MM_Official

The enrolled identity of a senior leader / official whose voice biometric
is captured. The "known-good baseline" of PS4 example #3.

```yaml
apiName: MMOfficial
primaryKey: id (string)
properties:
  id: string                 # required, PK, e.g. "off-vance-state"
  name: string               # required, e.g. "Secretary J.D. Vance"
  role: string               # e.g. "Secretary of State"
  photo_uri: string
  voice_embedding: string    # 192-dim float comma-joined; ~3KB
  enrollment_date: timestamp
  embedding_quality: double  # 0..1, proxy for source-audio SNR
  status: string             # "active" | "revoked"
```

### MM_VoiceBiometric

Per-enrollment audit record. Allows multiple embeddings per official
(re-enrollment, channel-specific embeddings, etc.). Optional in MVP.

```yaml
apiName: MMVoiceBiometric
primaryKey: id (string)
properties:
  id: string             # required, PK
  official_id: string    # FK -> MMOfficial.id
  embedding: string      # 192-dim float comma-joined
  sample_audio_uri: string
  quality_score: double
  created_at: timestamp
```

### MM_Communication

Each incoming voice / text / document / image event flowing into a
command-grade decision channel.

```yaml
apiName: MMCommunication
primaryKey: id (string)
properties:
  id: string                 # required, PK
  channel: string            # "radio" | "teams" | "zoom" | "voicememo" | "pdf" | "image" | "text"
  audio_uri: string          # nullable for non-audio
  transcript: string         # populated by Citadel ASR for audio
  sender_claimed: string     # what the metadata says (e.g. "From: SecState Vance")
  received_at: timestamp
```

### MM_AuthEvent

The verdict record for any single scan. One AuthEvent per
voice-auth check or guardrail scan. The forensic trail.

```yaml
apiName: MMAuthEvent
primaryKey: id (string)
properties:
  id: string                 # required, PK
  communication_id: string   # FK -> MMCommunication.id
  official_id: string        # FK -> MMOfficial.id; nullable when no enrolled match
  speaker_match: double      # 0..1, cosine similarity (audio events)
  deepfake_risk: integer     # 0..100 from Citadel
  injection_risk: integer    # 0..100 from Citadel text gateway
  verdict: string            # "VERIFIED" | "WARN" | "BLOCKED"
  latency_ms: integer
  signals_json: string       # full JSON of every signal returned by Citadel
  processed_at: timestamp
```

## Reused object types (not creating; already in ontology)

- **RouteAlert** — surfaced human-readable threat, has status workflow
- **RouteAlertComment** — audit-trail comments under each alert
- **Drones**, **Aircraft** — fleet-under-defense narrative for the C2 slide

## Action types

### mm-enroll-official

```yaml
apiName: mm-enroll-official
operation: createObject
targetType: MMOfficial
parameters: <maps to all MM_Official fields>
```

### mm-log-auth-event

```yaml
apiName: mm-log-auth-event
operation: createObject
targetType: MMAuthEvent
parameters: <maps to all MM_AuthEvent fields>
```

## Reused action types (already in ontology)

- `add-route-alert-comment...` — attach forensic comments
- `[Example] Update Route Alert Status` — escalate / close

## Why we store embeddings as joined strings

Foundry Workshop and Ontology Manager don't natively support `array<double>`
properties on object types at our tier. So we store the 192-dim float array
as a CSV string. The Go gateway and Python sidecar both serialize/deserialize
this format. Tradeoff: ~6x larger than raw bytes but introspectable in
Workshop tables.
