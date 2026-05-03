import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'

type Health = {
  status: string
  uptime_s?: number
  citadel_configured?: boolean
  foundry_configured?: boolean
  voicebio_url?: string
  enrolled_count?: number
} | null

type RegistryProfile = {
  id: string
  name: string
  role?: string
  sha512_fingerprint?: string
  public_key?: string
  status?: string
  registry_status?: string
  source?: string
  realness?: string
  enrolled_at?: string
  enrollment_center?: string
}

type AuthEvent = {
  id: string
  channel: string
  verdict: 'VERIFIED' | 'WARN' | 'BLOCKED'
  verdict_reason?: string
  sender_claimed?: string
  official_name?: string
  speaker_match?: number
  deepfake_risk?: number
  injection_risk?: number
  overall_risk?: number
  latency_ms?: number
  oob_response?: string
  oob_prompt_id?: string
  transcript?: string
  processed_at?: string
}

type AuthResponse = {
  event?: AuthEvent
  error?: string
}

type AuditEntry = {
  id: string
  type: string
  ts: string
  actor_id?: string
  fingerprint?: string
  prev_hash?: string
  hash?: string
  payload?: unknown
}

type OOBPrompt = {
  type: string
  prompt_id: string
  official_id: string
  nonce: string
  snippet?: string
  deadline?: string
  signals?: Record<string, unknown>
}

// IdentityEnvelope is the internal in-memory shape used by the signer/phone:
// id + public_key + private_key flattened. The wire format is split in two:
// PublicIdentityEnvelope (registry-safe) + EnrollmentSecret (one-time
// private-key delivery). flattenCredential() merges either form back.
type IdentityEnvelope = {
  id: string
  public_key: string
  private_key: string
}

type PublicIdentityEnvelope = {
  v?: string
  alg?: string
  id: string
  public_key: string
  registry_url?: string
  fingerprint?: string
  issued_at?: string
}

type EnrollmentSecret = {
  warning?: string
  id: string
  alg?: string
  private_key: string
  issued_at?: string
  post_quantum_roadmap?: string
  passkey_roadmap?: string
}

// Browser-local credential cache. After enrollment we automatically persist
// the credential bundle here so Sign + Phone don't need a re-upload of the
// downloaded file. The UI still offers download for cross-device portability.
const LOCAL_CREDENTIALS_KEY = 'mm.credentials.v1'

type StoredCredential = {
  id: string
  name?: string
  role?: string
  saved_at: string
  bundle: { identity_envelope: PublicIdentityEnvelope; enrollment_secret: EnrollmentSecret }
}

function loadStoredCredentials(): StoredCredential[] {
  try {
    const raw = window.localStorage.getItem(LOCAL_CREDENTIALS_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter((c: unknown) => {
      if (!c || typeof c !== 'object') return false
      const sc = c as Record<string, unknown>
      return typeof sc.id === 'string' && typeof sc.bundle === 'object'
    }) as StoredCredential[]
  } catch {
    return []
  }
}

function saveStoredCredential(entry: StoredCredential) {
  const existing = loadStoredCredentials().filter((c) => c.id !== entry.id)
  const next = [entry, ...existing].slice(0, 16)
  window.localStorage.setItem(LOCAL_CREDENTIALS_KEY, JSON.stringify(next))
}

function removeStoredCredential(id: string) {
  const next = loadStoredCredentials().filter((c) => c.id !== id)
  window.localStorage.setItem(LOCAL_CREDENTIALS_KEY, JSON.stringify(next))
}

function flattenCredential(raw: unknown): IdentityEnvelope | null {
  if (!raw || typeof raw !== 'object') return null
  const r = raw as Record<string, unknown>
  // New bundle form
  if (r.identity_envelope && r.enrollment_secret) {
    const env = r.identity_envelope as PublicIdentityEnvelope
    const sec = r.enrollment_secret as EnrollmentSecret
    if (!env.public_key || !sec.private_key) return null
    if (env.id && sec.id && env.id !== sec.id) return null
    return { id: env.id || sec.id, public_key: env.public_key, private_key: sec.private_key }
  }
  // Legacy flat form
  if (typeof r.id === 'string' && typeof r.public_key === 'string' && typeof r.private_key === 'string') {
    return { id: r.id, public_key: r.public_key, private_key: r.private_key }
  }
  return null
}

type DemoKind = 'real' | 'clone'
type EnrollResult = {
  profile?: RegistryProfile
  identity_envelope?: PublicIdentityEnvelope
  enrollment_secret?: EnrollmentSecret
  error?: string
}
type VerifyResult = {
  ok?: boolean
  official_id?: string
  official_name?: string
  status?: string
  signature_valid?: boolean
  audio_hash_match?: boolean
  voice_match?: boolean
  warning?: string
  error?: string
}
type CloneEvalSummary = {
  n?: number
  allow?: number
  warn?: number
  block?: number
  detected_warn_or_block?: number | null
}
type CloneEval = {
  created_at?: string
  policy?: string
  summary?: Record<string, CloneEvalSummary>
  rows?: Array<{
    label?: string
    file?: string
    action?: string
    risk_score?: number
    deepfake_signal?: number
    wavlm_signal?: number
    processing_time_ms?: number
  }>
}
type DemoFixtures = {
  policy?: string
  provider_counts?: Record<string, number>
}
type RecordedAudio = {
  blob: Blob
  filename: string
  url: string
}

const DEMO_OFFICIAL_ID = 'demo-official-093854'
const enrollmentScript = 'I am enrolling this voice as my live biometric baseline for Mighty Morphing Trust Gate. I understand this identity can sign media and approve sensitive voice requests.'
const guidance = {
  console: {
    what: 'Runs the end-to-end trust gate demo: registry identity, voice-risk checks, approval-device challenge, and signed allow/block decision.',
    how: 'Use the left rail from top to bottom: confirm the enrolled profile, load the approval-device identity, run a real fixture or controlled clone fixture, then read the decision stream.',
  },
  enroll: {
    what: 'Creates the trusted public record for a person: public key, voiceprint fingerprint, status, and server signature.',
    how: 'Use a unique ID, read the enrollment script aloud, record or upload the sample, then download the one-time identity envelope for signing and approval-device use.',
  },
  sign: {
    what: 'Turns generated or recorded audio into signed media that anyone can verify against the public registry.',
    how: 'Use the identity_envelope.json downloaded after enrollment. Pick the exact audio file, sign it, then download media.sig.json and keep it beside the audio.',
  },
  verify: {
    what: 'Checks whether an audio file is authentic, unchanged, active in the registry, and still matches the enrolled voice.',
    how: 'Search the public registry first to confirm who the signature should resolve to, then upload the audio and media.sig.json signature envelope.',
  },
  registry: {
    what: 'Provides the public, queryable trust layer that other systems can use to validate officials and signed media.',
    how: 'Search an official ID or open the shareable registry URL to inspect public key, voiceprint fingerprint, and status.',
  },
  audit: {
    what: 'Shows the tamper-evident record of enrollments, signatures, verifications, gateway decisions, and revocations.',
    how: 'Refresh after each scenario and confirm the audit chain remains valid.',
  },
  phone: {
    what: 'Acts as the approval device that prevents a cloned voice from approving its own command.',
    how: 'Load the private identity envelope downloaded during enrollment. Approve expected real-call prompts and deny controlled clone-fixture prompts.',
  },
}

const routes = [
  ['/', 'Trust Gate Demo'],
  ['/enroll', 'Enroll'],
  ['/sign', 'Sign'],
  ['/verify', 'Verify'],
  ['/registry', 'Registry'],
  ['/audit', 'Audit'],
  [`/phone/${DEMO_OFFICIAL_ID}`, 'Approval Device'],
]

export default function App() {
  const [path, setPath] = useState(window.location.pathname)
  const [health, setHealth] = useState<Health>(null)
  const [healthErr, setHealthErr] = useState<string | null>(null)

  useEffect(() => {
    fetch('/health')
      .then((r) => r.json())
      .then(setHealth)
      .catch((e: unknown) => setHealthErr(String(e)))
  }, [])

  useEffect(() => {
    const onPop = () => setPath(window.location.pathname)
    window.addEventListener('popstate', onPop)
    return () => window.removeEventListener('popstate', onPop)
  }, [])

  function navigate(next: string) {
    window.history.pushState(null, '', next)
    setPath(next)
  }

  const page = useMemo(() => {
    if (path === '/enroll') return <EnrollPage />
    if (path === '/sign') return <SignPage />
    if (path === '/verify') return <VerifyPage />
    if (path === '/registry') return <RegistryPage />
    if (path === '/audit') return <AuditPage />
    if (path.startsWith('/phone/')) return <PhonePage id={path.split('/').pop() ?? ''} />
    return <ConsolePage />
  }, [path])
  const activeGuide = guideForPath(path)

  return (
    <div className="soft-shell min-h-screen bg-zinc-950 text-zinc-100">
      <header className="border-b border-zinc-800 bg-zinc-950">
        <div className="mx-auto flex max-w-7xl flex-col gap-3 px-4 py-4 md:flex-row md:items-end md:justify-between">
          <div>
            <div className="text-xs font-semibold uppercase tracking-wide text-emerald-300">PS4 Digital Defense</div>
            <h1 className="mt-1 text-lg font-semibold">Mighty Morphing Trust Gate</h1>
            <div className="mt-1 max-w-3xl text-xs text-zinc-500">Stops cloned voices, replayed audio, and unsigned media from becoming trusted agent commands across high-impact workflows.</div>
          </div>
          <div className="flex flex-wrap items-center gap-2 text-xs">
            <StatusPill label="Gateway" ok={health?.status === 'ok'} value={healthErr ?? health?.status ?? '...'} />
            <StatusPill label="Citadel" ok={!!health?.citadel_configured} />
            <StatusPill label="Voicebio" ok={!!health?.voicebio_url} />
            <StatusPill label="Registry" ok={health?.enrolled_count !== undefined} value={String(health?.enrolled_count ?? 0)} />
          </div>
        </div>
        <nav className="mx-auto flex max-w-7xl gap-1 overflow-x-auto px-4 pb-3">
          {routes.map(([href, label]) => (
            <button
              key={href}
              type="button"
              onClick={() => navigate(href)}
              className={
                'shrink-0 border px-3 py-1.5 text-xs ' +
                (path === href || (href === '/' && path === '/console')
                  ? 'border-emerald-600 bg-emerald-950 text-emerald-200'
                  : 'border-zinc-800 bg-zinc-900 text-zinc-400 hover:border-zinc-700 hover:text-zinc-200')
              }
            >
              {label}
            </button>
          ))}
        </nav>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-5">
        <PageIntro what={activeGuide.what} how={activeGuide.how} />
        {page}
      </main>
    </div>
  )
}

function guideForPath(path: string) {
  if (path === '/enroll') return guidance.enroll
  if (path === '/sign') return guidance.sign
  if (path === '/verify') return guidance.verify
  if (path === '/registry') return guidance.registry
  if (path === '/audit') return guidance.audit
  if (path.startsWith('/phone/')) return guidance.phone
  return guidance.console
}

function PageIntro({ what, how }: { what: string; how: string }) {
  return (
    <section className="mb-4 border border-zinc-800 bg-zinc-950 p-4">
      <div className="grid gap-3 lg:grid-cols-2">
        <div>
          <div className="text-[10px] font-semibold uppercase tracking-wide text-emerald-300">What this does</div>
          <div className="mt-1 text-sm text-zinc-100">{what}</div>
        </div>
        <div>
          <div className="text-[10px] font-semibold uppercase tracking-wide text-sky-300">How to use it</div>
          <div className="mt-1 text-sm text-zinc-100">{how}</div>
        </div>
      </div>
    </section>
  )
}

function StatusPill({ label, ok, value }: { label: string; ok: boolean; value?: string }) {
  return (
    <span className={'border px-2 py-1 ' + (ok ? 'border-emerald-700 text-emerald-300' : 'border-zinc-800 text-zinc-500')}>
      {label}: {value ?? (ok ? 'ok' : 'off')}
    </span>
  )
}

function ConsolePage() {
  const [events, setEvents] = useState<AuthEvent[]>([])
  const [officials, setOfficials] = useState<RegistryProfile[]>([])
  const [demoBusy, setDemoBusy] = useState<DemoKind | null>(null)
  const [demoError, setDemoError] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<AuthEvent | null>(null)
  const [fixtures, setFixtures] = useState<DemoFixtures | null>(null)
  const [cloneEval, setCloneEval] = useState<CloneEval | null>(null)

  useEffect(() => {
    refresh()
    fetch('/demo/fixtures')
      .then((r) => r.json())
      .then(setFixtures)
      .catch(() => undefined)
    fetch('/demo/clone-eval')
      .then((r) => r.json())
      .then(setCloneEval)
      .catch(() => undefined)
    const es = new EventSource('/api/events')
    es.addEventListener('auth.event', (event) => {
      const rec = JSON.parse((event as MessageEvent).data) as AuthEvent
      setEvents((prev) => [rec, ...prev].slice(0, 50))
      setLastResult(rec)
    })
    es.addEventListener('official.enrolled', () => refresh())
    return () => es.close()
  }, [])

  function refresh() {
    fetch('/api/events/recent')
      .then((r) => r.json())
      .then((body) => setEvents(body.data ?? []))
      .catch(() => undefined)
    fetch('/registry/officials')
      .then((r) => r.json())
      .then((body) => setOfficials(body.data ?? []))
      .catch(() => undefined)
  }

  // Adapt to whatever's actually in the registry. Prefer the seeded demo
  // official if it exists, otherwise the most recently enrolled active record.
  const activeOfficial =
    officials.find((o) => o.id === DEMO_OFFICIAL_ID) ??
    officials.find((o) => o.status === 'active') ??
    officials[0] ??
    null

  async function runDemo(kind: DemoKind) {
    setDemoBusy(kind)
    setDemoError(null)
    if (!activeOfficial) {
      setDemoError('No active official enrolled. Open Enroll first to create one, then return here.')
      setDemoBusy(null)
      return
    }
    const form = new FormData()
    form.set('claimed_official_id', activeOfficial.id)
    form.set('channel', kind === 'real' ? 'verified-call' : 'clone-attack')
    try {
      const res = await fetch(`/demo/gateway/${kind}`, { method: 'POST', body: form })
      const body = (await res.json()) as AuthResponse
      if (!res.ok || body.error) throw new Error(body.error ?? `HTTP ${res.status}`)
      if (body.event) {
        setLastResult(body.event)
        setEvents((prev) => [body.event as AuthEvent, ...prev.filter((e) => e.id !== body.event?.id)].slice(0, 50))
      }
    } catch (e: unknown) {
      setDemoError(String(e))
    } finally {
      setDemoBusy(null)
    }
  }

  return (
    <div className="space-y-4">
      <ImpactSummary />
      <div className="grid gap-4 xl:grid-cols-[360px_1fr]">
        <div className="space-y-4">
          <DemoControls busy={demoBusy} error={demoError} official={activeOfficial} result={lastResult} onRun={runDemo} />
          <CloneDefenseLab fixtures={fixtures} evalResult={cloneEval} />
          <PhonePanel id={activeOfficial?.id ?? DEMO_OFFICIAL_ID} />
          <section className="border border-zinc-800 bg-zinc-950 p-4">
            <SectionTitle title="Registry" right={String(officials.length)} />
            <div className="space-y-2">
              {officials.length === 0 ? <Empty text="No profiles" /> : officials.map((o) => <ProfileRow key={o.id} profile={o} />)}
            </div>
          </section>
        </div>
        <section className="border border-zinc-800 bg-zinc-950 p-4">
          <SectionTitle title="Signed Decision Stream" right={String(events.length)} />
          <p className="mb-3 text-xs text-zinc-500">Each row is the gateway's final command-grade decision after registry lookup, passive signal checks, and approval-device response.</p>
          <div className="space-y-2">
            {events.length === 0 ? <Empty text="No events" /> : events.map((e) => <EventRow key={e.id} event={e} />)}
          </div>
        </section>
      </div>
    </div>
  )
}

function ImpactSummary() {
  return (
    <section className="grid gap-3 md:grid-cols-3">
      <ImpactCard label="Problem" value="Voice clones can authorize real-world agent actions." />
      <ImpactCard label="Boundary" value="Unsigned or unapproved audio never controls an agent." />
      <ImpactCard label="Scale" value="One registry lookup and one signed decision work across every agency, vendor, and workflow." />
    </section>
  )
}

function ImpactCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-zinc-800 bg-zinc-950 p-4">
      <div className="text-[10px] font-semibold uppercase tracking-wide text-zinc-500">{label}</div>
      <div className="mt-2 text-sm font-medium text-zinc-100">{value}</div>
    </div>
  )
}

function DemoControls({
  busy,
  error,
  official,
  result,
  onRun,
}: {
  busy: DemoKind | null
  error: string | null
  official?: RegistryProfile
  result: AuthEvent | null
  onRun: (kind: DemoKind) => void
}) {
  return (
    <section className="border border-zinc-800 bg-zinc-950 p-4">
      <SectionTitle title="Controlled Demo Fixtures" right={official?.status ?? 'profile'} />
      <div className="mb-3 border border-zinc-800 bg-zinc-900 p-3">
        <div className="text-sm font-medium text-zinc-100">{official?.name ?? 'No active official'}</div>
        <div className="mt-1 text-xs text-zinc-500">{official?.id ?? 'enroll on the Enroll tab to create one'}</div>
        <div className="mt-2 text-xs text-zinc-400">{profileTrustText(official)}</div>
      </div>
      <div className="grid gap-2">
        <button
          type="button"
          disabled={!!busy}
          onClick={() => onRun('real')}
          className="border border-emerald-700 bg-emerald-950 px-3 py-3 text-left text-sm text-emerald-100 disabled:opacity-50"
        >
          <div className="font-semibold">{busy === 'real' ? 'Waiting for phone' : 'Start real call'}</div>
          <div className="mt-1 text-xs text-emerald-300/80">Controlled fixture: expected result is APPROVE on the approval device, then VERIFIED.</div>
        </button>
        <button
          type="button"
          disabled={!!busy}
          onClick={() => onRun('clone')}
          className="border border-rose-700 bg-rose-950 px-3 py-3 text-left text-sm text-rose-100 disabled:opacity-50"
        >
          <div className="font-semibold">{busy === 'clone' ? 'Waiting for phone' : 'Run controlled clone simulation'}</div>
          <div className="mt-1 text-xs text-rose-300/80">Uses backend demo fixtures only. Expected result is DENY or fail closed, then BLOCKED.</div>
        </button>
      </div>
      {error ? <div className="mt-3 border border-rose-800 bg-rose-950 p-2 text-xs text-rose-200">{error}</div> : null}
      {result ? <DecisionSummary event={result} /> : null}
    </section>
  )
}

function DecisionSummary({ event }: { event: AuthEvent }) {
  const label = decisionLabel(event)
  return (
    <div className="mt-3 border border-zinc-800 p-3">
      <div className="flex items-center justify-between gap-3">
        <div className="text-xs uppercase text-zinc-500">Last decision</div>
        <VerdictBadge verdict={event.verdict} />
      </div>
      <div className="mt-2 text-sm font-medium text-zinc-100">{label}</div>
      <div className="mt-1 text-xs text-zinc-500">{event.verdict_reason ?? event.channel}</div>
    </div>
  )
}

function CloneDefenseLab({ fixtures, evalResult }: { fixtures: DemoFixtures | null; evalResult: CloneEval | null }) {
  const counts = fixtures?.provider_counts ?? {}
  const summary = evalResult?.summary ?? {}
  const rows = (evalResult?.rows ?? []).slice(0, 8)
  return (
    <section className="border border-zinc-800 bg-zinc-950 p-4">
      <SectionTitle title="Clone Defense Lab" right={evalResult?.created_at ? 'evaluated' : 'pending'} />
      <div className="mb-3 text-xs text-zinc-400">
        Controlled fixtures only: user-owned real recordings, Cartesia clone fixtures, and ElevenLabs benchmark artifacts when present.
      </div>
      <div className="grid gap-2 text-xs sm:grid-cols-3">
        <MetricTile label="Real holdouts" value={String(counts.real ?? 0)} />
        <MetricTile label="Cartesia clones" value={String(counts.cartesia ?? 0)} />
        <MetricTile label="ElevenLabs artifacts" value={String(counts.elevenlabs ?? 0)} />
      </div>
      <div className="mt-3 grid gap-2">
        {Object.entries(summary).map(([label, item]) => (
          <div key={label} className="border border-zinc-800 bg-zinc-900/60 p-3">
            <div className="flex items-center justify-between gap-3">
              <div className="text-sm font-medium text-zinc-100">{formatEvalLabel(label)}</div>
              <div className="font-mono text-xs text-zinc-400">{item.detected_warn_or_block ?? '-'} / {item.n ?? 0} detected</div>
            </div>
            <div className="mt-2 grid grid-cols-3 gap-2 text-xs">
              <Metric label="allow" value={String(item.allow ?? 0)} />
              <Metric label="warn" value={String(item.warn ?? 0)} />
              <Metric label="block" value={String(item.block ?? 0)} />
            </div>
          </div>
        ))}
        {Object.keys(summary).length === 0 ? <Empty text="Run clone-defense evaluation to populate live model scores." /> : null}
      </div>
      {rows.length ? (
        <div className="mt-3 space-y-2">
          {rows.map((row) => (
            <div key={`${row.label}-${row.file}`} className="flex flex-wrap items-center justify-between gap-2 border border-zinc-800 bg-zinc-900/40 p-2 text-xs">
              <span className="truncate text-zinc-300">{row.file?.split('/').pop()}</span>
              <span className="font-mono text-zinc-400">{row.action ?? '-'} risk={row.risk_score ?? '-'} df={row.deepfake_signal ?? '-'} wl={row.wavlm_signal ?? '-'}</span>
            </div>
          ))}
        </div>
      ) : null}
      {fixtures?.policy ? <div className="mt-3 text-[11px] text-zinc-500">{fixtures.policy}</div> : null}
      {evalResult?.policy ? <div className="mt-1 text-[11px] text-zinc-500">{evalResult.policy}</div> : null}
    </section>
  )
}

function EnrollPage() {
  const [result, setResult] = useState<EnrollResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [recordedAudio, setRecordedAudio] = useState<RecordedAudio | null>(null)
  const [id, setId] = useState(() => uniqueEnrollmentID())
  const [idStatus, setIDStatus] = useState<string | null>(null)
  const identity = result?.identity_envelope ?? null
  const secret = result?.enrollment_secret ?? null
  const credentialBundle = identity && secret
    ? { identity_envelope: identity, enrollment_secret: secret }
    : null
  const credentialFilename = identity
    ? `${identity.id}.credential.json`
    : 'mighty-morphing.credential.json'

  useEffect(() => {
    const trimmed = id.trim()
    if (!trimmed) {
      setIDStatus('ID is required and must be unique.')
      return
    }
    let cancelled = false
    fetch(`/registry/officials/${encodeURIComponent(trimmed)}`)
      .then((res) => {
        if (cancelled) return
        setIDStatus(res.ok ? 'This ID already exists. Use a unique ID before enrolling.' : 'ID is available for this enrollment.')
      })
      .catch(() => {
        if (!cancelled) setIDStatus('Uniqueness will be checked by the registry when you enroll.')
      })
    return () => {
      cancelled = true
    }
  }, [id])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setResult(null)
    const form = new FormData(event.currentTarget)
    if (recordedAudio) {
      form.set('audio', new File([recordedAudio.blob], recordedAudio.filename, { type: recordedAudio.blob.type }))
    }
    try {
      const res = await fetch('/registry/enroll', { method: 'POST', body: form })
      const body: EnrollResult = await res.json()
      setResult(body)
      if (body.identity_envelope && body.enrollment_secret) {
        saveStoredCredential({
          id: body.identity_envelope.id,
          name: body.profile?.name,
          role: body.profile?.role,
          saved_at: new Date().toISOString(),
          bundle: {
            identity_envelope: body.identity_envelope,
            enrollment_secret: body.enrollment_secret,
          },
        })
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <FormShell title="Enrollment Center">
      <FlowSteps
        steps={[
          'Create a unique official ID. Do not reuse the demo ID unless intentionally replacing a local demo profile.',
          `Read aloud: "${enrollmentScript}"`,
          'Record from the microphone or upload the same clean audio file.',
          'Enroll, then download the credential file. The public registry stores only the public key and voiceprint fingerprint; the credential file is the only place your private signing key is ever delivered.',
        ]}
      />
      <ScriptCard title="Enrollment script" text={enrollmentScript} />
      <AudioRecorder label="Live voice sample" onReady={setRecordedAudio} />
      <form onSubmit={submit} className="grid gap-3 md:grid-cols-2">
        <TextInput name="id" label="Unique official ID" required value={id} onChange={setId} />
        <TextInput name="name" label="Name" required />
        <TextInput name="role" label="Role or authority" defaultValue="controlled demo official" />
        <TextInput name="enrollment_center" label="Enrollment source" defaultValue="local controlled demo enrollment" />
        <FileInput name="audio" label={recordedAudio ? 'Audio fallback' : 'Audio file'} accept="audio/*" />
        <SubmitButton busy={busy} label="Enroll" />
      </form>
      {idStatus ? <div className={'mt-3 border p-2 text-xs ' + (idStatus.includes('already') ? 'border-rose-800 bg-rose-950 text-rose-200' : 'border-zinc-800 bg-zinc-900 text-zinc-300')}>{idStatus}</div> : null}
      {credentialBundle && secret ? (
        <>
          <ResultPanel tone="success" title="Enrollment created">
            <div className="space-y-2 text-sm text-emerald-200">
              <div>The public registry now stores your public key and voiceprint fingerprint. The credential — including your private signing key — has been cached in this browser so Sign and Approval Device can use it without a re-upload.</div>
              <div className="text-emerald-300/80">Optional: download the credential file if you want to use this identity from a different browser or device.</div>
              <button className="mt-1 block w-full border border-emerald-700 bg-emerald-950 px-3 py-2 text-sm text-emerald-100 hover:bg-emerald-900" onClick={() => downloadJSON(credentialFilename, credentialBundle)}>
                Download {credentialFilename} (portable copy)
              </button>
              <div className="mt-2 text-xs text-emerald-300/80">→ Go to <span className="font-semibold">Sign</span> to issue a signed media envelope using this identity.</div>
            </div>
          </ResultPanel>
          <div className="mt-3 border border-rose-700 bg-rose-950 p-3 text-xs text-rose-100">
            <div className="font-semibold uppercase tracking-wide text-rose-200">Private key — one-time delivery</div>
            <div className="mt-1">{secret.warning}</div>
            <div className="mt-2 text-rose-300/80">
              <div><span className="text-rose-200">Algorithm:</span> {secret.alg ?? 'ed25519'}</div>
              {secret.post_quantum_roadmap ? <div className="mt-1"><span className="text-rose-200">PQ roadmap:</span> {secret.post_quantum_roadmap}</div> : null}
              {secret.passkey_roadmap ? <div className="mt-1"><span className="text-rose-200">Passkey roadmap:</span> {secret.passkey_roadmap}</div> : null}
            </div>
          </div>
        </>
      ) : null}
      <JSONBlock value={result} />
    </FormShell>
  )
}

function SignPage() {
  const [result, setResult] = useState<{ signature?: unknown; error?: string } | null>(null)
  const [busy, setBusy] = useState(false)
  const [recordedAudio, setRecordedAudio] = useState<RecordedAudio | null>(null)
  const [storedCredentials, setStoredCredentials] = useState<StoredCredential[]>(() => loadStoredCredentials())
  const [selectedCredentialID, setSelectedCredentialID] = useState<string>(() => loadStoredCredentials()[0]?.id ?? '')
  const [credentialFile, setCredentialFile] = useState<File | null>(null)
  const [uploadedCredentialID, setUploadedCredentialID] = useState<string | null>(null)
  const [credentialError, setCredentialError] = useState<string | null>(null)
  const [purpose, setPurpose] = useState('mighty-morphing.media')
  const [audioFile, setAudioFile] = useState<File | null>(null)
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null)
  const [verifyBusy, setVerifyBusy] = useState(false)
  const signature = result?.signature ?? null
  const signatureBlob = signature ? new Blob([JSON.stringify(signature)], { type: 'application/json' }) : null
  const activeCredential = storedCredentials.find((c) => c.id === selectedCredentialID) ?? null

  function refreshStoredCredentials() {
    const list = loadStoredCredentials()
    setStoredCredentials(list)
    if (list.length === 0) setSelectedCredentialID('')
    else if (!list.some((c) => c.id === selectedCredentialID)) setSelectedCredentialID(list[0].id)
  }

  async function onCredentialPicked(event: FormEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0] ?? null
    setCredentialFile(file)
    setUploadedCredentialID(null)
    setCredentialError(null)
    if (!file) return
    try {
      const raw = JSON.parse(await file.text())
      const flat = flattenCredential(raw)
      if (!flat) {
        setCredentialError('Credential file missing identity_envelope or enrollment_secret.')
        return
      }
      setUploadedCredentialID(flat.id)
    } catch {
      setCredentialError('Credential file is not valid JSON.')
    }
  }

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setResult(null)
    setVerifyResult(null)
    if (!recordedAudio && !audioFile) {
      setResult({ error: 'Record or pick an audio file to sign.' })
      return
    }
    setBusy(true)
    const form = new FormData()
    if (credentialFile) {
      form.set('identity', credentialFile)
    } else if (activeCredential) {
      form.set(
        'identity_envelope',
        JSON.stringify({
          identity_envelope: activeCredential.bundle.identity_envelope,
          enrollment_secret: activeCredential.bundle.enrollment_secret,
        }),
      )
    } else {
      setBusy(false)
      setResult({ error: 'No credential available. Enroll first or upload a credential file.' })
      return
    }
    if (recordedAudio) {
      form.set('audio', new File([recordedAudio.blob], recordedAudio.filename, { type: recordedAudio.blob.type }))
    } else if (audioFile) {
      form.set('audio', audioFile)
    }
    form.set('purpose', purpose)
    try {
      const res = await fetch('/sign', { method: 'POST', body: form })
      setResult(await res.json())
    } finally {
      setBusy(false)
    }
  }

  // Negative test: verify the just-signed bundle, but swap the audio for the
  // built-in canonical clone fixture. Cryptographic signature still verifies
  // (it is over the original audio hash), but the audio_hash_match check
  // fails — proving an attacker cannot reuse the signature on substituted
  // audio. If the verifier can also resolve the issuer, voice_match will
  // surface the speaker mismatch separately.
  async function runStolenAudioReplay() {
    if (!signatureBlob) return
    setVerifyBusy(true)
    setVerifyResult(null)
    try {
      const fixtureRes = await fetch('/demo/clone-eval')
      const fixturePayload = await fixtureRes.json().catch(() => null)
      const fixtureRow: { file?: string } | undefined = fixturePayload?.rows?.find((r: { file?: string }) => typeof r.file === 'string')
      let attackerBlob: Blob | null = null
      let attackerName = 'cloned_or_substituted.wav'
      if (fixtureRow?.file) {
        try {
          const ab = await fetch(`/demo/fixtures/audio?path=${encodeURIComponent(fixtureRow.file)}`)
          if (ab.ok) {
            attackerBlob = await ab.blob()
            attackerName = fixtureRow.file.split('/').pop() ?? attackerName
          }
        } catch {
          // fall through
        }
      }
      if (!attackerBlob) {
        // Fall back: synthesize a quick PCM of silence + tone so the hash differs.
        const sampleRate = 16000
        const seconds = 1.5
        const samples = Math.floor(sampleRate * seconds)
        const pcm = new Int16Array(samples)
        for (let i = 0; i < samples; i++) pcm[i] = Math.floor(Math.sin(i / 16) * 4096)
        attackerBlob = new Blob([pcm.buffer], { type: 'audio/L16' })
        attackerName = 'attacker-substitute.raw'
      }
      const form = new FormData()
      form.set('audio', new File([attackerBlob], attackerName, { type: attackerBlob.type }))
      form.set('sig', new File([signatureBlob], 'media.sig.json', { type: 'application/json' }))
      const res = await fetch('/verify', { method: 'POST', body: form })
      setVerifyResult(await res.json())
    } finally {
      setVerifyBusy(false)
    }
  }

  return (
    <FormShell title="Generation Signing">
      <FlowSteps
        steps={[
          'Start on Enroll and download the credential file after a successful enrollment.',
          'Record live audio here, or pick a file you want others to verify.',
          'Pick the credential file. The signer reads the private key locally and the registry resolves the public key.',
          'After signing, run the stolen-audio replay to prove a captured signature cannot be reused on substituted audio.',
        ]}
      />
      <AudioRecorder label="Audio to sign (live)" onReady={setRecordedAudio} />
      <div className="mb-3 border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-300">
        <div className="mb-1 font-semibold uppercase tracking-wide text-zinc-200">Demo fixtures</div>
        <div className="mb-2 text-zinc-400">Load a controlled audio fixture to demonstrate the layered defenses without leaving the UI. Cartesia clones were generated offline with consent; the UI never calls a cloning provider.</div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className="border border-emerald-700 bg-emerald-950 px-3 py-2 text-emerald-100 hover:bg-emerald-900"
            onClick={async () => {
              try {
                const res = await fetch('/demo/fixture-audio?kind=real')
                if (!res.ok) return
                const blob = await res.blob()
                const filename = res.headers.get('X-Fixture-Filename') ?? 'real_fixture.wav'
                setAudioFile(new File([blob], filename, { type: blob.type || 'audio/wav' }))
                setRecordedAudio(null)
              } catch {
                /* ignore */
              }
            }}
          >
            Load real-speaker fixture
          </button>
          <button
            type="button"
            className="border border-rose-700 bg-rose-950 px-3 py-2 text-rose-100 hover:bg-rose-900"
            onClick={async () => {
              try {
                const res = await fetch('/demo/fixture-audio?kind=clone')
                if (!res.ok) return
                const blob = await res.blob()
                const filename = res.headers.get('X-Fixture-Filename') ?? 'clone_cartesia_10.wav'
                setAudioFile(new File([blob], filename, { type: blob.type || 'audio/wav' }))
                setRecordedAudio(null)
              } catch {
                /* ignore */
              }
            }}
          >
            Load Cartesia clone fixture
          </button>
        </div>
        {audioFile ? <div className="mt-2 text-[11px] text-emerald-400">Loaded: {audioFile.name} ({Math.round(audioFile.size / 1024)} KB)</div> : null}
      </div>
      <form onSubmit={submit} className="grid gap-3 md:grid-cols-2">
        <label className="grid gap-1 text-sm text-zinc-400">
          Audio file fallback
          <input
            type="file"
            accept="audio/*"
            onChange={(e) => setAudioFile(e.currentTarget.files?.[0] ?? null)}
            className="border border-zinc-800 bg-zinc-900 px-3 py-2 file:mr-3 file:border-0 file:bg-zinc-800 file:px-2 file:py-1 file:text-zinc-200"
          />
          {recordedAudio ? <span className="text-[11px] text-emerald-400">Live recording will be used; this fallback is ignored.</span> : null}
        </label>
        <label className="grid gap-1 text-sm text-zinc-400">
          Sign as
          {storedCredentials.length > 0 ? (
            <select
              value={selectedCredentialID}
              onChange={(e) => {
                setSelectedCredentialID(e.currentTarget.value)
                setCredentialFile(null)
                setUploadedCredentialID(null)
              }}
              className="border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-emerald-700"
            >
              {storedCredentials.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name ? `${c.name} (${c.id})` : c.id}
                </option>
              ))}
            </select>
          ) : (
            <div className="border border-zinc-800 bg-zinc-900 px-3 py-2 text-xs text-zinc-500">No credentials cached. Enroll first, or upload a credential file below.</div>
          )}
          <div className="mt-1 flex items-center gap-2 text-[11px]">
            {activeCredential ? (
              <span className="text-emerald-400">Signing as: {activeCredential.id}</span>
            ) : null}
            {storedCredentials.length > 0 ? (
              <button
                type="button"
                onClick={() => {
                  if (!selectedCredentialID) return
                  removeStoredCredential(selectedCredentialID)
                  refreshStoredCredentials()
                }}
                className="ml-auto text-rose-400 underline hover:text-rose-300"
              >
                Forget this credential on this browser
              </button>
            ) : null}
          </div>
        </label>
        <label className="grid gap-1 text-sm text-zinc-400">
          Or upload a credential file (cross-device)
          <input
            type="file"
            accept="application/json,.json"
            onChange={onCredentialPicked}
            className="border border-zinc-800 bg-zinc-900 px-3 py-2 file:mr-3 file:border-0 file:bg-zinc-800 file:px-2 file:py-1 file:text-zinc-200"
          />
          {uploadedCredentialID ? <span className="text-[11px] text-emerald-400">Uploaded credential for id: {uploadedCredentialID} (overrides the dropdown selection)</span> : null}
          {credentialError ? <span className="text-[11px] text-rose-400">{credentialError}</span> : null}
        </label>
        <TextInput name="purpose" label="Purpose" value={purpose} onChange={setPurpose} />
        <SubmitButton busy={busy} label="Sign" />
      </form>
      {signature ? (
        <ResultPanel tone="success" title="Detached signature ready">
          <div className="space-y-2 text-sm text-emerald-200">
            <div>Distribute this file beside the audio. The verifier will resolve the public key from the registry — no private material crosses the wire.</div>
            <button className="block border border-emerald-700 bg-emerald-950 px-3 py-2 text-sm text-emerald-100 hover:bg-emerald-900" onClick={() => downloadJSON('media.sig.json', signature)}>Download media.sig.json</button>
            <div className="mt-3 border-t border-emerald-900 pt-3 text-xs text-emerald-300">
              <div className="font-semibold uppercase tracking-wide text-emerald-200">Negative test — substitute the audio</div>
              <div className="mt-1">Replays this signature against a different audio clip. A cryptographically valid signature is bound to the original audio hash, so the verifier must reject the swap.</div>
              <button
                type="button"
                onClick={runStolenAudioReplay}
                disabled={verifyBusy}
                className="mt-2 border border-rose-700 bg-rose-950 px-3 py-2 text-sm text-rose-100 hover:bg-rose-900 disabled:opacity-40"
              >
                {verifyBusy ? 'Replaying...' : 'Run stolen-audio replay'}
              </button>
            </div>
          </div>
        </ResultPanel>
      ) : null}
      {verifyResult ? <NegativeTestSummary result={verifyResult} /> : null}
      <JSONBlock value={result} />
    </FormShell>
  )
}

function NegativeTestSummary({ result }: { result: VerifyResult }) {
  const blocked = result.audio_hash_match === false || result.signature_valid === false || result.voice_match === false
  const tone: 'success' | 'warn' = blocked ? 'success' : 'warn'
  const title = blocked ? 'Replay attack blocked' : 'Replay attack PASSED — investigate'
  const failures = [
    result.audio_hash_match === false ? 'audio hash does not match the signed payload' : null,
    result.signature_valid === false ? 'signature does not match the registry public key' : null,
    result.voice_match === false ? 'voiceprint does not match the issuer in the registry' : null,
  ].filter(Boolean) as string[]
  return (
    <ResultPanel tone={tone} title={title}>
      {blocked ? (
        <div className="space-y-1 text-sm">
          <div>Verifier rejected the substituted audio:</div>
          <ul className="ml-5 list-disc text-xs">
            {failures.map((f) => <li key={f}>{f}</li>)}
          </ul>
          <div className="mt-1 text-xs text-zinc-400">This is the layered defense: cryptographic identity catches forged signatures, audio hash catches substitution, voiceprint catches cloned-voice replay.</div>
        </div>
      ) : (
        <div className="text-sm">Verifier accepted the substituted audio — this should never happen with the layered checks. Check the verify endpoint configuration.</div>
      )}
    </ResultPanel>
  )
}

function VerifyPage() {
  const [result, setResult] = useState<VerifyResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [profiles, setProfiles] = useState<RegistryProfile[]>([])
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<RegistryProfile | null>(null)

  useEffect(() => {
    fetch('/registry/officials')
      .then((r) => r.json())
      .then((body) => setProfiles(body.data ?? []))
      .catch(() => undefined)
  }, [])

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setBusy(true)
    setResult(null)
    const form = new FormData(event.currentTarget)
    try {
      const res = await fetch('/verify', { method: 'POST', body: form })
      setResult(await res.json())
    } finally {
      setBusy(false)
    }
  }

  return (
    <FormShell title="Standalone Verification">
      <section className="mb-4 grid gap-3 lg:grid-cols-[320px_1fr]">
        <div className="border border-zinc-800 bg-zinc-900 p-3">
          <SectionTitle title="Public Registry Search" right={String(filteredProfiles(profiles, query).length)} />
          <input
            value={query}
            onChange={(event) => setQuery(event.currentTarget.value)}
            placeholder="Search name, role, or official ID"
            className="w-full border border-zinc-800 bg-zinc-950 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-emerald-700"
          />
          <div className="mt-3 max-h-72 space-y-2 overflow-auto">
            {filteredProfiles(profiles, query).map((profile) => (
              <button key={profile.id} type="button" onClick={() => setSelected(profile)} className="w-full text-left">
                <ProfileRow profile={profile} />
              </button>
            ))}
            {profiles.length === 0 ? <Empty text="Registry not loaded. Verification still works with audio plus signature." /> : null}
          </div>
        </div>
        <div className="border border-zinc-800 bg-zinc-900 p-3">
          <SectionTitle title="Expected Public Identity" right={selected?.status ?? 'select'} />
          {selected ? (
            <div className="space-y-2">
              <RegistryField label="Who this should verify as" value={`${selected.name} (${selected.id})`} />
              <RegistryField label="Trust explanation" value={profileTrustText(selected)} />
              <RegistryField label="Public lookup URL" value={`${window.location.origin}/registry/officials/${selected.id}`} mono />
            </div>
          ) : (
            <Empty text="Select a registry entry to make the expected signer explicit before uploading files." />
          )}
        </div>
      </section>
      <form onSubmit={submit} className="grid gap-3 md:grid-cols-2">
        <FileInput name="audio" label="Audio being verified" accept="audio/*" required />
        <FileInput name="sig" label="media.sig.json signature envelope" accept="application/json,.json" required />
        <SubmitButton busy={busy} label="Verify" />
      </form>
      {result ? <VerifySummary result={result} /> : null}
      <JSONBlock value={result} />
    </FormShell>
  )
}

function RegistryPage() {
  const [profiles, setProfiles] = useState<RegistryProfile[]>([])
  // Empty default search so the registry page shows EVERY enrolled official by
  // default. Pre-filling DEMO_OFFICIAL_ID made it look like only one record
  // existed even when the registry had several.
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<RegistryProfile | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    fetch('/registry/officials')
      .then((r) => r.json())
      .then((body) => {
        const list = (body.data ?? []) as RegistryProfile[]
        setProfiles(list)
        setSelected(list.find((profile) => profile.id === DEMO_OFFICIAL_ID) ?? list[0] ?? null)
      })
      .catch((e: unknown) => setError(String(e)))
  }, [])

  async function lookup(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setSelected(null)
    try {
      const trimmed = query.trim()
      if (!trimmed) {
        const res = await fetch('/registry/officials')
        const body = await res.json()
        if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`)
        const list = (body.data ?? []) as RegistryProfile[]
        setProfiles(list)
        setSelected(list[0] ?? null)
        return
      }
      const res = await fetch(`/registry/search?q=${encodeURIComponent(trimmed)}&limit=20`)
      const body = await res.json()
      if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`)
      const list = (body.data ?? []) as RegistryProfile[]
      if (list.length === 0) throw new Error('No public registry entry matched this search.')
      setProfiles(list)
      setSelected(list[0])
    } catch (e: unknown) {
      setError(String(e))
    }
  }

  const publicURL = selected ? `${window.location.origin}/registry/officials/${selected.id}` : `${window.location.origin}/registry/officials/{official_id}`
  const visibleProfiles = filteredProfiles(profiles, query)

  return (
    <section className="grid gap-4 lg:grid-cols-[360px_1fr]">
      <div className="border border-zinc-800 bg-zinc-950 p-4">
        <SectionTitle title="Registry Lookup" right={String(profiles.length)} />
        <form onSubmit={lookup} className="grid gap-2">
          <label className="grid gap-1 text-xs text-zinc-400">
            Search or exact official ID
            <input value={query} onChange={(event) => setQuery(event.currentTarget.value)} className="border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-emerald-700" />
          </label>
          <button className="border border-emerald-700 bg-emerald-950 px-3 py-2 text-sm text-emerald-100">Search public registry</button>
        </form>
        <div className="mt-3 border border-zinc-800 bg-zinc-900 p-2 text-xs text-zinc-400">Search resolves against the public registry metadata: official ID, name, role, key fingerprint, status, source, and realness label.</div>
        {error ? <div className="mt-3 border border-rose-800 bg-rose-950 p-2 text-xs text-rose-200">{error}</div> : null}
        <div className="mt-4 space-y-2">
          {visibleProfiles.map((profile) => (
            <button key={profile.id} type="button" onClick={() => { setSelected(profile); setQuery(profile.id) }} className="w-full text-left">
              <ProfileRow profile={profile} />
            </button>
          ))}
          {visibleProfiles.length === 0 ? <Empty text="No registry entries match this search" /> : null}
        </div>
      </div>
      <div className="border border-zinc-800 bg-zinc-950 p-4">
        <SectionTitle title="Public Validation Record" right={selected?.status ?? 'waiting'} />
        {selected ? (
          <div className="space-y-3">
            <RegistryField label="Trusted network URL" value={publicURL} />
            <RegistryField label="Official" value={`${selected.name} (${selected.id})`} />
            <RegistryField label="Realness and source" value={profileTrustText(selected)} />
            <RegistryField label="Public key" value={selected.public_key ?? '-'} mono />
            <RegistryField label="Voiceprint fingerprint" value={selected.sha512_fingerprint ?? '-'} mono />
            <RegistryField label="Status" value={selected.status ?? 'unknown'} />
            <RegistryField label="Enrollment source" value={selected.enrollment_center ?? 'not published'} />
            <RegistryField label="Enrolled at" value={selected.enrolled_at ?? 'not published'} />
          </div>
        ) : (
          <Empty text="No profile selected" />
        )}
      </div>
    </section>
  )
}

function RegistryField({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="border border-zinc-800 p-3">
      <div className="text-[10px] font-semibold uppercase tracking-wide text-zinc-500">{label}</div>
      <div className={(mono ? 'font-mono ' : '') + 'mt-1 break-all text-sm text-zinc-100'}>{value}</div>
    </div>
  )
}

function AuditPage() {
  const [body, setBody] = useState<{ events?: AuditEntry[]; chain_valid?: boolean } | null>(null)

  function refresh() {
    fetch('/audit')
      .then((r) => r.json())
      .then(setBody)
      .catch((e: unknown) => setBody({ events: [], chain_valid: false, error: String(e) } as never))
  }

  useEffect(refresh, [])

  return (
    <section className="border border-zinc-800 bg-zinc-950 p-4">
      <div className="mb-3 flex items-center justify-between">
        <SectionTitle title="Audit Log" right={body?.chain_valid ? 'valid' : 'invalid'} />
        <button type="button" className="border border-zinc-700 px-3 py-1.5 text-xs" onClick={refresh}>Refresh</button>
      </div>
      <div className="space-y-2">
        {(body?.events ?? []).map((event) => (
          <div key={event.id} className="border border-zinc-800 p-3">
            <div className="flex flex-wrap justify-between gap-2">
              <span className="font-mono text-xs text-zinc-300">{event.type}</span>
              <span className="font-mono text-xs text-zinc-500">{event.ts}</span>
            </div>
            <div className="mt-2 break-all font-mono text-[11px] text-zinc-500">{event.hash}</div>
          </div>
        ))}
      </div>
      <JSONBlock value={body} />
    </section>
  )
}

function PhonePage({ id }: { id: string }) {
  return (
    <div className="mx-auto max-w-md">
      <PhonePanel id={id} />
    </div>
  )
}

function PhonePanel({ id }: { id: string }) {
  const [status, setStatus] = useState('load identity')
  const [prompt, setPrompt] = useState<OOBPrompt | null>(null)
  const [socket, setSocket] = useState<WebSocket | null>(null)
  const [identity, setIdentity] = useState<IdentityEnvelope | null>(null)
  const storageKey = `mm.identity.${id}`

  useEffect(() => {
    const raw = window.localStorage.getItem(storageKey)
    if (!raw) return
    try {
      const parsed = flattenCredential(JSON.parse(raw))
      if (parsed && parsed.id === id) {
        setIdentity(parsed)
        setStatus('connecting')
      } else {
        window.localStorage.removeItem(storageKey)
      }
    } catch {
      window.localStorage.removeItem(storageKey)
    }
  }, [id, storageKey])

  useEffect(() => {
    if (!identity) return
    const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const ws = new WebSocket(`${scheme}//${window.location.host}/ws/client?official_id=${encodeURIComponent(id)}`)
    setSocket(ws)
    ws.onopen = () => setStatus('connected')
    ws.onclose = () => setStatus('disconnected')
    ws.onerror = () => setStatus('error')
    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data) as { type?: string; error?: string; challenge?: string }
      const challenge = msg.challenge
      if (msg.type === 'oob.challenge' && challenge) {
        signChallenge(identity, challenge, id)
          .then((signature) => ws.send(JSON.stringify({ type: 'oob.challenge_response', challenge, signature })))
          .catch((e: unknown) => setStatus(`challenge failed: ${String(e)}`))
      }
      if (msg.type === 'oob.prompt') setPrompt(msg as OOBPrompt)
      if (msg.type === 'oob.error') setStatus((msg as { error?: string }).error ?? 'error')
    }
    return () => ws.close()
  }, [id, identity])

  async function loadIdentity(event: FormEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0]
    if (!file) return
    let raw: unknown
    try {
      raw = JSON.parse(await file.text())
    } catch {
      setStatus('credential file is not valid JSON')
      return
    }
    const parsed = flattenCredential(raw)
    if (!parsed) {
      setStatus('credential file missing identity_envelope or enrollment_secret')
      return
    }
    if (parsed.id !== id) {
      setStatus('wrong identity')
      return
    }
    window.localStorage.setItem(storageKey, JSON.stringify(raw))
    setIdentity(parsed)
    setStatus('connecting')
  }

  function respond(verdict: 'APPROVE' | 'DENY') {
    if (!socket || !prompt) return
    socket.send(JSON.stringify({ type: 'oob.response', prompt_id: prompt.prompt_id, verdict }))
    setPrompt(null)
  }

  function clearIdentity() {
    window.localStorage.removeItem(storageKey)
    setIdentity(null)
    setPrompt(null)
    setStatus('load identity')
    socket?.close()
  }

  return (
    <section className="border border-zinc-800 bg-zinc-950 p-4">
      <SectionTitle title="Approval Device" right={status} />
      <div className="mb-3 border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-400">
        This is the simulated second device for out-of-band approval. It uses the private identity envelope from Enroll to answer server challenges, then lets the operator approve real prompts or deny controlled clone-fixture prompts.
      </div>
      {!identity ? (
        <label className="grid gap-2 text-sm text-zinc-400">
          identity_envelope.json
          <input type="file" accept="application/json,.json" onChange={loadIdentity} className="border border-zinc-800 bg-zinc-900 px-3 py-2 file:mr-3 file:border-0 file:bg-zinc-800 file:px-2 file:py-1 file:text-zinc-200" />
        </label>
      ) : prompt ? (
        <div className="space-y-3 border border-zinc-800 p-3">
          <div className="text-xs uppercase text-zinc-500">{prompt.official_id}</div>
          <div className="text-sm text-zinc-200">{prompt.snippet || 'Inbound voice verification request'}</div>
          <div className="text-xs text-zinc-500">Approve only if this is the expected real-call fixture. Deny controlled clone simulations and anything unexpected.</div>
          <div className="grid grid-cols-2 gap-2">
            <button className="border border-emerald-700 bg-emerald-950 px-3 py-3 text-emerald-100" onClick={() => respond('APPROVE')}>Approve</button>
            <button className="border border-rose-700 bg-rose-950 px-3 py-3 text-rose-100" onClick={() => respond('DENY')}>Deny</button>
          </div>
        </div>
      ) : (
        <div className="border border-zinc-800 p-3">
          <div className="text-sm text-zinc-200">Ready for challenge</div>
          <div className="mt-1 break-all font-mono text-[11px] text-zinc-500">{id}</div>
          <button type="button" onClick={clearIdentity} className="mt-3 border border-zinc-700 px-2 py-1 text-xs text-zinc-400">Reset key</button>
        </div>
      )}
    </section>
  )
}

function AudioRecorder({ label, onReady }: { label: string; onReady: (audio: RecordedAudio) => void }) {
  const recorderRef = useRef<MediaRecorder | null>(null)
  const chunksRef = useRef<Blob[]>([])
  const streamRef = useRef<MediaStream | null>(null)
  const [recording, setRecording] = useState(false)
  const [audio, setAudio] = useState<RecordedAudio | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    return () => {
      streamRef.current?.getTracks().forEach((track) => track.stop())
      if (audio?.url) URL.revokeObjectURL(audio.url)
    }
  }, [audio?.url])

  async function start() {
    setError(null)
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const recorder = new MediaRecorder(stream)
    streamRef.current = stream
    recorderRef.current = recorder
    chunksRef.current = []
    recorder.ondataavailable = (event) => {
      if (event.data.size > 0) chunksRef.current.push(event.data)
    }
    recorder.onstop = () => {
      stream.getTracks().forEach((track) => track.stop())
      const mime = recorder.mimeType || 'audio/webm'
      const ext = mime.includes('ogg') ? 'ogg' : mime.includes('mp4') ? 'm4a' : 'webm'
      const blob = new Blob(chunksRef.current, { type: mime })
      const next = { blob, filename: `live-sample.${ext}`, url: URL.createObjectURL(blob) }
      setAudio(next)
      onReady(next)
      setRecording(false)
    }
    recorder.start()
    setRecording(true)
  }

  function stop() {
    recorderRef.current?.stop()
  }

  async function toggle() {
    try {
      if (recording) stop()
      else await start()
    } catch (e: unknown) {
      setError(String(e))
      setRecording(false)
    }
  }

  return (
    <div className="mb-4 border border-zinc-800 bg-zinc-900 p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-xs font-semibold uppercase tracking-wide text-zinc-500">{label}</div>
          <div className="mt-1 text-sm text-zinc-200">{audio ? audio.filename : recording ? 'Recording now' : 'Record from microphone'}</div>
        </div>
        <button type="button" onClick={toggle} className={(recording ? 'border-rose-700 bg-rose-950 text-rose-100' : 'border-emerald-700 bg-emerald-950 text-emerald-100') + ' border px-3 py-2 text-sm'}>
          {recording ? 'Stop' : 'Record'}
        </button>
      </div>
      {audio ? <audio className="mt-3 w-full" controls src={audio.url} /> : null}
      {error ? <div className="mt-2 text-xs text-rose-300">{error}</div> : null}
    </div>
  )
}

function FlowSteps({ steps }: { steps: string[] }) {
  return (
    <div className="mb-4 grid gap-2 md:grid-cols-2">
      {steps.map((step, index) => (
        <div key={step} className="border border-zinc-800 bg-zinc-900 p-3">
          <div className="text-[10px] font-semibold uppercase tracking-wide text-emerald-300">Step {index + 1}</div>
          <div className="mt-1 text-sm text-zinc-200">{step}</div>
        </div>
      ))}
    </div>
  )
}

function ScriptCard({ title, text }: { title: string; text: string }) {
  return (
    <div className="mb-4 border border-sky-800 bg-sky-950/40 p-3">
      <div className="text-[10px] font-semibold uppercase tracking-wide text-sky-300">{title}</div>
      <div className="mt-1 text-sm text-sky-100">{text}</div>
    </div>
  )
}

function ResultPanel({ tone, title, children }: { tone: 'success' | 'warn'; title: string; children: ReactNode }) {
  const style = tone === 'success' ? 'border-emerald-800 bg-emerald-950/40 text-emerald-100' : 'border-amber-800 bg-amber-950/40 text-amber-100'
  return (
    <div className={'mt-4 border p-3 text-sm ' + style}>
      <div className="font-semibold">{title}</div>
      <div className="mt-1">{children}</div>
    </div>
  )
}

function VerifySummary({ result }: { result: VerifyResult }) {
  const ok = !!result.ok
  return (
    <ResultPanel tone={ok ? 'success' : 'warn'} title={ok ? 'Verification passed' : 'Verification did not pass'}>
      <div className="grid gap-2 text-xs md:grid-cols-4">
        <Metric label="signature" value={flagText(result.signature_valid)} />
        <Metric label="audio hash" value={flagText(result.audio_hash_match)} />
        <Metric label="registry status" value={result.status ?? '-'} />
        <Metric label="voice match" value={flagText(result.voice_match)} />
      </div>
      <div className="mt-2 text-xs">{result.warning ?? result.error ?? `${result.official_name ?? result.official_id ?? 'Signer'} resolved through the public registry.`}</div>
    </ResultPanel>
  )
}

async function signChallenge(identity: IdentityEnvelope, challenge: string, expectedID: string): Promise<string> {
  if (identity.id && identity.id !== expectedID) {
    throw new Error('identity id does not match phone route')
  }
  const rawPrivate = hexToBytes(identity.private_key)
  if (rawPrivate.length !== 64) throw new Error('expected 64-byte Ed25519 private key')
  const seed = rawPrivate.slice(0, 32)
  const pkcs8Prefix = hexToBytes('302e020100300506032b657004220420')
  const pkcs8 = concatBytes(pkcs8Prefix, seed)
  const key = await crypto.subtle.importKey('pkcs8', toArrayBuffer(pkcs8), { name: 'Ed25519' } as AlgorithmIdentifier, false, ['sign'])
  const sig = await crypto.subtle.sign({ name: 'Ed25519' } as AlgorithmIdentifier, key, new TextEncoder().encode(challenge))
  return bytesToBase64(new Uint8Array(sig))
}

function hexToBytes(hex: string): Uint8Array {
  const clean = hex.trim()
  if (clean.length % 2 !== 0) throw new Error('invalid hex')
  const out = new Uint8Array(clean.length / 2)
  for (let i = 0; i < out.length; i += 1) out[i] = Number.parseInt(clean.slice(i * 2, i * 2 + 2), 16)
  return out
}

function concatBytes(a: Uint8Array, b: Uint8Array): Uint8Array {
  const out = new Uint8Array(a.length + b.length)
  out.set(a, 0)
  out.set(b, a.length)
  return out
}

function bytesToBase64(bytes: Uint8Array): string {
  let binary = ''
  for (const byte of bytes) binary += String.fromCharCode(byte)
  return btoa(binary)
}

function toArrayBuffer(bytes: Uint8Array): ArrayBuffer {
  const out = new ArrayBuffer(bytes.byteLength)
  new Uint8Array(out).set(bytes)
  return out
}

function FormShell({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="border border-zinc-800 bg-zinc-950 p-4">
      <SectionTitle title={title} />
      {children}
    </section>
  )
}

function TextInput({
  name,
  label,
  required,
  defaultValue,
  value,
  onChange,
}: {
  name: string
  label: string
  required?: boolean
  defaultValue?: string
  value?: string
  onChange?: (value: string) => void
}) {
  return (
    <label className="grid gap-1 text-xs text-zinc-400">
      {label}
      <input
        name={name}
        required={required}
        defaultValue={value === undefined ? defaultValue : undefined}
        value={value}
        onChange={onChange ? (event) => onChange(event.currentTarget.value) : undefined}
        className="border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-emerald-700"
      />
    </label>
  )
}

function FileInput({ name, label, accept, required }: { name: string; label: string; accept?: string; required?: boolean }) {
  return (
    <label className="grid gap-1 text-xs text-zinc-400">
      {label}
      <input name={name} required={required} type="file" accept={accept} className="border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 file:mr-3 file:border-0 file:bg-zinc-800 file:px-2 file:py-1 file:text-zinc-200" />
    </label>
  )
}

function SubmitButton({ busy, label }: { busy: boolean; label: string }) {
  return (
    <div className="flex items-end">
      <button disabled={busy} className="w-full border border-emerald-700 bg-emerald-950 px-3 py-2 text-sm text-emerald-100 disabled:opacity-50">
        {busy ? 'Working' : label}
      </button>
    </div>
  )
}

function SectionTitle({ title, right }: { title: string; right?: string }) {
  return (
    <div className="mb-3 flex items-baseline justify-between">
      <h2 className="text-xs font-semibold uppercase tracking-wide text-zinc-400">{title}</h2>
      {right ? <span className="text-xs text-zinc-500">{right}</span> : null}
    </div>
  )
}

function Empty({ text }: { text: string }) {
  return <div className="border border-dashed border-zinc-800 p-4 text-sm text-zinc-500">{text}</div>
}

function ProfileRow({ profile }: { profile: RegistryProfile }) {
  return (
    <div className="border border-zinc-800 p-3">
      <div className="flex items-baseline justify-between gap-3">
        <div className="font-medium">{profile.name}</div>
        <div className="text-xs text-zinc-500">{profile.registry_status ?? profile.status ?? 'unknown'}</div>
      </div>
      <div className="mt-1 text-xs text-zinc-400">{profile.id}{profile.role ? ` | ${profile.role}` : ''}</div>
      <div className="mt-2 flex flex-wrap gap-2 text-[11px] text-zinc-400">
        <span className="border border-zinc-800 px-2 py-1">{profile.realness ?? 'realness-unpublished'}</span>
        <span className="border border-zinc-800 px-2 py-1">{profile.source ?? profile.enrollment_center ?? 'source-unpublished'}</span>
      </div>
      {profile.sha512_fingerprint ? <div className="mt-2 break-all font-mono text-[11px] text-zinc-500">{profile.sha512_fingerprint}</div> : null}
    </div>
  )
}

function EventRow({ event }: { event: AuthEvent }) {
  const style = event.verdict === 'VERIFIED' ? 'border-emerald-800' : event.verdict === 'BLOCKED' ? 'border-rose-800' : 'border-amber-800'
  return (
    <div className={'border p-3 ' + style}>
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <div>
          <VerdictBadge verdict={event.verdict} />
          <div className="mt-2 text-sm font-medium text-zinc-100">{decisionLabel(event)}</div>
        </div>
        <div className="font-mono text-xs opacity-70">{event.latency_ms ?? 0}ms</div>
      </div>
      <div className="mt-1 text-xs text-zinc-500">{event.verdict_reason ?? event.channel}</div>
      <div className="mt-3 grid gap-2 text-xs md:grid-cols-4">
        <Metric label="speaker match" value={scoreText(event.speaker_match)} />
        <Metric label="synthetic risk" value={riskText(event.deepfake_risk)} />
        <Metric label="injection risk" value={riskText(event.injection_risk)} />
        <Metric label="phone" value={event.oob_response ?? '-'} />
      </div>
    </div>
  )
}

function VerdictBadge({ verdict }: { verdict: AuthEvent['verdict'] }) {
  const style = verdict === 'VERIFIED' ? 'border-emerald-700 bg-emerald-950 text-emerald-200' : verdict === 'BLOCKED' ? 'border-rose-700 bg-rose-950 text-rose-200' : 'border-amber-700 bg-amber-950 text-amber-200'
  return <span className={'inline-flex border px-2 py-1 text-xs font-semibold ' + style}>{verdict}</span>
}

function decisionLabel(event: AuthEvent) {
  if (event.verdict === 'VERIFIED') return 'Control allowed'
  if (event.oob_response === 'DENY') return 'Stopped by official'
  if (event.oob_response === 'TIMEOUT') return 'Stopped without phone approval'
  if (event.verdict === 'BLOCKED') return 'Control blocked'
  return 'Needs review'
}

function scoreText(value?: number) {
  if (value === undefined) return '-'
  return `${Math.round(value * 100)}%`
}

function riskText(value?: number) {
  if (value === undefined) return '-'
  return `${value}/100`
}

function formatEvalLabel(label: string) {
  return label
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

function flagText(value?: boolean) {
  if (value === undefined) return '-'
  return value ? 'pass' : 'fail'
}

function uniqueEnrollmentID() {
  const stamp = new Date().toISOString().replace(/[-:.TZ]/g, '').slice(0, 14)
  const suffix = Math.random().toString(36).slice(2, 8)
  return `official-${stamp}-${suffix}`
}

function filteredProfiles(profiles: RegistryProfile[], query: string) {
  const needle = query.trim().toLowerCase()
  if (!needle) return profiles
  return profiles.filter((profile) =>
    [
      profile.id,
      profile.name,
      profile.role,
      profile.status,
      profile.registry_status,
      profile.source,
      profile.realness,
      profile.enrollment_center,
      profile.sha512_fingerprint,
      profile.public_key,
    ].some((value) => value?.toLowerCase().includes(needle)),
  )
}

function profileTrustText(profile?: RegistryProfile | null) {
  if (!profile) return 'No public registry record loaded yet. The demo can still run if backend fixtures are seeded.'
  const publishedStatus = profile.registry_status ?? profile.status ?? 'unknown'
  const status = publishedStatus === 'active' ? 'active public registry entry' : `${publishedStatus} registry entry`
  const source = profile.source ?? profile.enrollment_center ?? 'source not published'
  const realness = profile.realness ?? 'realness not published'
  const name = profile.name || profile.id
  if (profile.id === DEMO_OFFICIAL_ID) return `${name} is the seeded controlled demo identity, not a live government attestation. Status: ${status}. Source: ${source}. Realness label: ${realness}.`
  return `${name} is treated according to its published registry status, key, fingerprint, enrollment source, and realness label. Status: ${status}. Source: ${source}. Realness label: ${realness}.`
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-[10px] uppercase text-zinc-500">{label}</div>
      <div className="font-mono text-sm">{value}</div>
    </div>
  )
}

function MetricTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-zinc-800 bg-zinc-900/60 p-3">
      <div className="text-[10px] uppercase text-zinc-500">{label}</div>
      <div className="mt-1 font-mono text-lg text-zinc-100">{value}</div>
    </div>
  )
}

function JSONBlock({ value }: { value: unknown }) {
  if (!value) return null
  return <pre className="mt-4 max-h-[420px] overflow-auto border border-zinc-800 bg-zinc-900 p-3 text-xs text-zinc-300">{JSON.stringify(value, null, 2)}</pre>
}

function downloadJSON(filename: string, value: unknown) {
  const blob = new Blob([JSON.stringify(value, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
