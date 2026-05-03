# Foundry Ontology Setup (one-time, ~10 min)

The Foundry Platform API at our hackathon tier does **not** allow programmatic
object-type creation (`POST /api/v2/ontologies/{rid}/objectTypes` returns 404).
So you create object types and action types once in the Ontology Manager UI;
after that, the Go gateway uses the API to create *instances* of those objects
and *apply* actions, which both work.

This is a **one-time** ~10-minute setup. Skip if you've already done it.

## Stack info (already configured)

- Stack: `https://nshackathon.palantirfoundry.com`
- Ontology: **NatSec Hackathon Ontology**
- RID: `ri.ontology.main.ontology.41fccd0c-2180-4c1d-841d-8a488d1abb46`

## Step 1 — open Ontology Manager

1. Go to `https://nshackathon.palantirfoundry.com`
2. Click the **app launcher** (top-left grid icon) → **Ontology Manager**
3. Click `NatSec Hackathon Ontology` to enter it
4. In the left sidebar, click **Object types**

## Step 2 — create the 4 object types

Click **`+ New`** (top-right) for each. Set:

### `MM_Official`

- API name: `MMOfficial`
- Display name: `MM Official`
- Plural: `MM Officials`
- Description: "Senior leader / official whose voice biometric is enrolled."
- **Primary key:** `id` (string)
- **Properties:**
  - `id` — string (PK, required)
  - `name` — string (required)
  - `role` — string (e.g., "Secretary of State")
  - `photo_uri` — string
  - `voice_embedding` — string (we store the 192-float vector as a comma-joined string; long; ~3KB)
  - `enrollment_date` — timestamp
  - `embedding_quality` — double
  - `status` — string ("active" / "revoked")

### `MM_VoiceBiometric`

- API name: `MMVoiceBiometric`
- Display name: `MM Voice Biometric`
- Plural: `MM Voice Biometrics`
- **Primary key:** `id` (string)
- **Properties:**
  - `id` — string (PK)
  - `official_id` — string (FK to MM_Official.id)
  - `embedding` — string (192-float comma-joined)
  - `sample_audio_uri` — string
  - `quality_score` — double
  - `created_at` — timestamp

### `MM_Communication`

- API name: `MMCommunication`
- Display name: `MM Communication`
- Plural: `MM Communications`
- **Primary key:** `id` (string)
- **Properties:**
  - `id` — string (PK)
  - `channel` — string (radio / teams / zoom / voicememo / pdf / image / text)
  - `audio_uri` — string
  - `transcript` — string
  - `sender_claimed` — string (the claimed identity from headers/metadata)
  - `received_at` — timestamp

### `MM_AuthEvent`

- API name: `MMAuthEvent`
- Display name: `MM Auth Event`
- Plural: `MM Auth Events`
- **Primary key:** `id` (string)
- **Properties:**
  - `id` — string (PK)
  - `communication_id` — string (FK to MM_Communication.id)
  - `official_id` — string (FK to MM_Official.id, nullable for unmatched)
  - `speaker_match` — double (0..1)
  - `deepfake_risk` — integer (0..100)
  - `injection_risk` — integer (0..100)
  - `verdict` — string (VERIFIED / WARN / BLOCKED)
  - `latency_ms` — integer
  - `signals_json` — string (raw JSON of all signal scores)
  - `processed_at` — timestamp

After saving each, **click "Edit" → "Save & Submit for review"** (or the
auto-publish equivalent in your Foundry tier). Or in dev/sandbox, the
publish button finalizes it immediately.

## Step 3 — create the 2 action types

Left sidebar → **Action types** → **`+ New`**.

### `mm-enroll-official`

- Display name: `MM Enroll Official`
- Operation type: **createObject**
- Target object type: `MMOfficial`
- Parameters: map all `MM_Official` properties as input parameters (`id`,
  `name`, `role`, `photo_uri`, `voice_embedding`, `enrollment_date`,
  `embedding_quality`, `status`).
- Effect: creates an `MMOfficial` with those values.

### `mm-log-auth-event`

- Display name: `MM Log Auth Event`
- Operation type: **createObject**
- Target object type: `MMAuthEvent`
- Parameters: map all `MM_AuthEvent` properties as input parameters.
- Effect: creates an `MMAuthEvent`.

## Step 4 — verify via API

After publishing, run:

```bash
set -a; source .env; set +a
curl -s -H "Authorization: Bearer $FOUNDRY_TOKEN" \
  "$FOUNDRY_STACK_URL/api/v2/ontologies/$FOUNDRY_ONTOLOGY_RID/objectTypes?pageSize=50" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); [print(o['apiName']) for o in d['data']]" \
  | grep -E "MM"
```

You should see:
```
MMOfficial
MMVoiceBiometric
MMCommunication
MMAuthEvent
```

And:
```bash
curl -s -H "Authorization: Bearer $FOUNDRY_TOKEN" \
  "$FOUNDRY_STACK_URL/api/v2/ontologies/$FOUNDRY_ONTOLOGY_RID/actionTypes?pageSize=50" \
  | python3 -c "import sys,json; d=json.load(sys.stdin); [print(a['apiName']) for a in d['data']]" \
  | grep -E "mm-"
```

You should see `mm-enroll-official` and `mm-log-auth-event`.

## Step 5 — done

The Go gateway now calls:
- `POST /api/v2/ontologies/{rid}/actions/mm-enroll-official/apply`
- `POST /api/v2/ontologies/{rid}/actions/mm-log-auth-event/apply`
- `POST /api/v2/ontologies/{rid}/actions/add-route-alert-comment.../apply` (existing)
- `POST /api/v2/ontologies/{rid}/actions/.../apply` (existing examples)

No further setup needed for the demo path.
