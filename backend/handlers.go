package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type server struct {
	cfg         config
	voice       *voiceClient
	citadel     *citadelClient
	foundry     *foundryClient
	sse         *sseHub
	enrollments *enrollmentStore
	signing     *signingKey
	events      *eventStore
	audit       *auditStore
	oob         *oobManager
	webauthn    *webauthnService
}

const defaultDemoOfficialID = "demo-official-093854"

// ============================================================================
// /api/enroll — voice → embedding → Official record (signed)
// ============================================================================

// enrollResponse splits the user's signing credential into a clearly-public
// identity envelope (matches what the registry stores) and a clearly-marked
// enrollmentSecret containing the one-time private key delivery. The legacy
// `identity_envelope` field with the private key inside is intentionally
// removed — clients must read the secret from `enrollment_secret` and store
// it separately.
type enrollResponse struct {
	Official         OfficialPublic          `json:"official"`
	IdentityEnvelope *publicIdentityEnvelope `json:"identity_envelope,omitempty"`
	EnrollmentSecret *enrollmentSecret       `json:"enrollment_secret,omitempty"`
	SignedEnvelope   *signedEnvelope         `json:"signed_envelope"`
	LatencyMs        int                     `json:"latency_ms"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
	ID    string `json:"id,omitempty"`
	Hint  string `json:"hint,omitempty"`
}

func (s *server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	id := strings.TrimSpace(r.FormValue("id"))
	name := strings.TrimSpace(r.FormValue("name"))
	role := strings.TrimSpace(r.FormValue("role"))
	photo := strings.TrimSpace(r.FormValue("photo_uri"))
	enrollmentCenter := strings.TrimSpace(r.FormValue("enrollment_center"))
	if id == "" || name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id and name are required"})
		return
	}
	identity, err := newIdentityEnvelope(id, registryBaseURL(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "audio file required: " + err.Error()})
		return
	}
	defer file.Close()
	audioBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	embed, err := s.voice.Embed(ctx, audioBytes, header.Filename)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "voicebio: " + err.Error()})
		return
	}

	off := &Official{
		ID:               id,
		Name:             name,
		Role:             role,
		PhotoURI:         photo,
		EnrollmentCenter: enrollmentCenter,
		PublicKey:        identity.PublicKey,
		EmbeddingQuality: embed.Quality,
		SampleDurationS:  embed.DurationS,
		Embedding:        embed.Embedding,
	}
	if err := s.enrollments.Add(off); err != nil {
		status := http.StatusInternalServerError
		var dup duplicateEnrollmentError
		if errors.As(err, &dup) {
			status = http.StatusConflict
			writeJSON(w, status, errorResponse{
				Error: err.Error(),
				Code:  "official_id_already_active",
				ID:    dup.ID,
				Hint:  "Use a unique id, or revoke the existing active registry record before re-enrolling this id.",
			})
			return
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	pub := off.Public()
	env, err := s.signing.Sign(pub)
	if err != nil {
		slog.Warn("sign enrollment envelope", "err", err)
	}
	if s.audit != nil {
		if _, err := s.audit.Append("enrollment.created", id, off.VoiceprintSHA512, off.RegistryPublic()); err != nil {
			slog.Warn("append audit enrollment", "err", err)
		}
	}

	s.sse.Broadcast("official.enrolled", pub)

	writeJSON(w, http.StatusOK, enrollResponse{
		Official:         pub,
		IdentityEnvelope: identity.Public(),
		EnrollmentSecret: identity.Secret(),
		SignedEnvelope:   env,
		LatencyMs:        int(time.Since(t0).Milliseconds()),
	})
}

// ============================================================================
// /api/officials — list / get / revoke
// ============================================================================

func (s *server) handleListOfficials(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data":   s.enrollments.List(),
		"issuer": s.signing.FingerprintShort(),
		"pubkey": s.signing.PublicKeyHex(),
	})
}

func (s *server) handleGetOfficial(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, ok := s.enrollments.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	writeJSON(w, http.StatusOK, o.Public())
}

func (s *server) handleRevokeOfficial(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.enrollments.Revoke(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	o, _ := s.enrollments.Get(id)
	pub := o.Public()
	if s.audit != nil {
		if _, err := s.audit.Append("identity.revoked", id, o.VoiceprintSHA512, o.RegistryPublic()); err != nil {
			slog.Warn("append audit revocation", "err", err)
		}
	}
	s.sse.Broadcast("official.revoked", pub)
	writeJSON(w, http.StatusOK, pub)
}

// ============================================================================
// /registry/* — public Layer 1 identity registry
// ============================================================================

type registryEnrollResponse struct {
	Profile          OfficialRegistryPublic  `json:"profile"`
	IdentityEnvelope *publicIdentityEnvelope `json:"identity_envelope"`
	EnrollmentSecret *enrollmentSecret       `json:"enrollment_secret,omitempty"`
	SignedEnvelope   *signedEnvelope         `json:"signed_envelope,omitempty"`
	LatencyMs        int                     `json:"latency_ms"`
}

func (s *server) handleRegistryEnroll(w http.ResponseWriter, r *http.Request) {
	// Same multipart contract as /api/enroll; the response includes the
	// one-time private identity envelope. Public reads stay on /registry/officials.
	s.handleEnroll(w, r)
}

func (s *server) handleRegistryOfficials(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	realness := strings.TrimSpace(r.URL.Query().Get("realness"))
	if q != "" || status != "" || source != "" || realness != "" || r.URL.Query().Get("limit") != "" {
		s.handleRegistrySearch(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data":            s.enrollments.RegistryList(),
		"issuer":          s.signing.FingerprintShort(),
		"pubkey":          s.signing.PublicKeyHex(),
		"metadata_schema": registryMetadataSchema(),
	})
}

func (s *server) handleRegistryOfficial(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, ok := s.enrollments.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	writeJSON(w, http.StatusOK, o.RegistryPublic())
}

func (s *server) handleRegistrySearch(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r.URL.Query().Get("limit"), 100)
	writeJSON(w, http.StatusOK, map[string]any{
		"data": s.enrollments.RegistrySearch(
			r.URL.Query().Get("q"),
			r.URL.Query().Get("status"),
			r.URL.Query().Get("source"),
			r.URL.Query().Get("realness"),
			limit,
		),
		"issuer":          s.signing.FingerprintShort(),
		"pubkey":          s.signing.PublicKeyHex(),
		"metadata_schema": registryMetadataSchema(),
	})
}

func (s *server) handleRegistryLookup(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	fingerprint := r.URL.Query().Get("fingerprint")
	publicKey := r.URL.Query().Get("public_key")
	if strings.TrimSpace(id) == "" && strings.TrimSpace(fingerprint) == "" && strings.TrimSpace(publicKey) == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{
			Error: "provide id, fingerprint, or public_key",
			Code:  "lookup_key_required",
		})
		return
	}
	profile, ok := s.enrollments.RegistryLookup(id, fingerprint, publicKey)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "registry entry not found", Code: "registry_entry_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profile":         profile,
		"issuer":          s.signing.FingerprintShort(),
		"pubkey":          s.signing.PublicKeyHex(),
		"metadata_schema": registryMetadataSchema(),
	})
}

func (s *server) handleRegistryIdentity(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	o, ok := s.enrollments.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "no such official", Code: "registry_entry_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"identity": publicIdentityEnvelope{
			V:           identityEnvelopeVersion,
			Alg:         "ed25519",
			ID:          o.ID,
			PublicKey:   o.PublicKey,
			RegistryURL: registryBaseURL(r) + "/officials/" + o.ID,
			Fingerprint: shortSHA512Hex([]byte(o.PublicKey)),
			IssuedAt:    o.EnrollmentDate.UTC().Format(time.RFC3339Nano),
		},
		"profile":               o.RegistryPublic(),
		"private_key_available": false,
	})
}

func registryMetadataSchema() map[string]any {
	return map[string]any{
		"source": map[string]string{
			"local-enrollment": "Created by this demo registry through the enrollment flow.",
		},
		"realness": map[string]string{
			"enrolled-real-speaker": "Registry metadata says the entry came from an enrollment recording; it is not a cryptographic proof that every future clip is real.",
		},
		"status": []string{"active", "revoked"},
	}
}

func (s *server) handleOOBWS(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("official_id"))
	off, ok := s.enrollments.Get(id)
	if id == "" || !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "official not enrolled"})
		return
	}
	if off.Status != "active" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "official is not active"})
		return
	}
	s.oob.ServeWSAuthenticated(w, r, id, off.PublicKey)
}

// ============================================================================
// /sign + /verify — standalone signed-media protocol
// ============================================================================

type signMediaResponse struct {
	Signature mediaSignatureEnvelope `json:"signature"`
	Audit     *auditEntry            `json:"audit,omitempty"`
}

func (s *server) handleSignMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	audioBytes, filename, err := readMultipartFile(r, "audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	identity, err := identityFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	off, ok := s.enrollments.Get(identity.ID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "issuer not found in registry"})
		return
	}
	if off.Status != "active" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "issuer is not active"})
		return
	}
	pubFromPrivate, err := publicKeyFromPrivateHex(identity.PrivateKey)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !strings.EqualFold(pubFromPrivate, off.PublicKey) || !strings.EqualFold(identity.PublicKey, off.PublicKey) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "identity envelope key does not match registry public key"})
		return
	}

	purpose := strings.TrimSpace(r.FormValue("purpose"))
	sig, err := signMediaEnvelope(identity.ID, off.VoiceprintSHA512, purpose, audioBytes, identity.PrivateKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	var audit *auditEntry
	if s.audit != nil {
		entry, err := s.audit.Append("media.signed", identity.ID, off.VoiceprintSHA512, map[string]any{
			"issuer_id":    identity.ID,
			"audio_sha512": sig.AudioSHA512,
			"purpose":      sig.Purpose,
			"filename":     filename,
		})
		if err != nil {
			slog.Warn("append audit media signed", "err", err)
		} else {
			audit = &entry
		}
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+identity.ID+`.sig.json"`)
	writeJSON(w, http.StatusOK, signMediaResponse{Signature: *sig, Audit: audit})
}

type verifyMediaResponse struct {
	OK             bool                    `json:"ok"`
	SignatureValid bool                    `json:"signature_valid"`
	AudioHashMatch bool                    `json:"audio_hash_match"`
	VoiceMatch     *bool                   `json:"voice_match,omitempty"`
	SpeakerCosine  *float64                `json:"speaker_cosine,omitempty"`
	Status         string                  `json:"status"`
	Fingerprint    string                  `json:"fingerprint,omitempty"`
	SignedAt       string                  `json:"signed_at,omitempty"`
	Issuer         *OfficialRegistryPublic `json:"issuer,omitempty"`
	Warning        string                  `json:"warning,omitempty"`
	Error          string                  `json:"error,omitempty"`
	Audit          *auditEntry             `json:"audit,omitempty"`
}

func (s *server) handleVerifyMedia(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	audioBytes, filename, err := readMultipartFile(r, "audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	sigBytes, _, err := readMultipartFile(r, "sig")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var sig mediaSignatureEnvelope
	if err := json.Unmarshal(sigBytes, &sig); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decode signature envelope: " + err.Error()})
		return
	}
	res := verifyMediaResponse{
		Status:      "unknown",
		Fingerprint: sig.Fingerprint,
		SignedAt:    sig.Timestamp,
	}
	off, ok := s.enrollments.Get(sig.IssuerID)
	if !ok {
		res.Error = "issuer not found in registry"
		writeJSON(w, http.StatusOK, res)
		return
	}
	registry := off.RegistryPublic()
	res.Issuer = &registry
	res.Status = off.Status

	hashMatch, sigValid, verifyErr := verifyMediaEnvelope(&sig, audioBytes, off.PublicKey)
	res.AudioHashMatch = hashMatch
	res.SignatureValid = sigValid
	if verifyErr != nil {
		res.Error = verifyErr.Error()
	}
	if sig.Fingerprint != "" && off.VoiceprintSHA512 != "" && !strings.EqualFold(sig.Fingerprint, off.VoiceprintSHA512) {
		res.Warning = "signature fingerprint differs from current registry voiceprint"
	}

	if sigValid && hashMatch && off.Status == "active" {
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		embed, embedErr := s.voice.Embed(ctx, audioBytes, filename)
		if embedErr != nil {
			res.Warning = strings.TrimSpace(res.Warning + "; speaker check unavailable: " + embedErr.Error())
		} else {
			cos := CosineSimilarity(off.Embedding, embed.Embedding)
			match := cos >= s.cfg.SpeakerMatchThreshold
			res.SpeakerCosine = &cos
			res.VoiceMatch = &match
			if !match {
				res.Warning = strings.TrimSpace(res.Warning + "; signature valid but voice does not match enrolled speaker")
			}
		}
	}
	voiceOK := res.VoiceMatch == nil || *res.VoiceMatch
	res.OK = res.SignatureValid && res.AudioHashMatch && res.Status == "active" && voiceOK

	if s.audit != nil {
		entry, err := s.audit.Append("media.verified", sig.IssuerID, off.VoiceprintSHA512, map[string]any{
			"issuer_id":        sig.IssuerID,
			"audio_sha512":     sig.AudioSHA512,
			"filename":         filename,
			"signature_valid":  res.SignatureValid,
			"audio_hash_match": res.AudioHashMatch,
			"voice_match":      res.VoiceMatch,
			"ok":               res.OK,
		})
		if err != nil {
			slog.Warn("append audit media verified", "err", err)
		} else {
			res.Audit = &entry
		}
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if s.audit == nil {
		writeJSON(w, http.StatusOK, map[string]any{"events": []auditEntry{}, "chain_valid": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"events":      s.audit.List(),
		"chain_valid": s.audit.ChainValid(),
		"issuer":      s.signing.FingerprintShort(),
		"pubkey":      s.signing.PublicKeyHex(),
	})
}

func identityFromRequest(r *http.Request) (*identityEnvelope, error) {
	if raw := strings.TrimSpace(r.FormValue("identity_envelope")); raw != "" {
		return parseIdentityEnvelope(raw)
	}
	if rawBytes, _, err := readMultipartFile(r, "identity"); err == nil {
		return parseIdentityEnvelope(string(rawBytes))
	}
	id := strings.TrimSpace(r.FormValue("id"))
	privateKey := strings.TrimSpace(r.FormValue("private_key"))
	publicKey := strings.TrimSpace(r.FormValue("public_key"))
	if id == "" || privateKey == "" {
		return nil, errors.New("provide identity file, identity_envelope JSON, or id + private_key")
	}
	if publicKey == "" {
		derived, err := publicKeyFromPrivateHex(privateKey)
		if err != nil {
			return nil, err
		}
		publicKey = derived
	}
	return &identityEnvelope{
		V:          identityEnvelopeVersion,
		Alg:        "ed25519",
		ID:         id,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

func readMultipartFile(r *http.Request, field string) ([]byte, string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return nil, "", fmt.Errorf("%s file required: %w", field, err)
	}
	defer file.Close()
	b, err := io.ReadAll(file)
	if err != nil {
		return nil, "", err
	}
	name := ""
	if header != nil {
		name = header.Filename
	}
	return b, name, nil
}

func registryBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		host = forwarded
	}
	if host == "" {
		host = "localhost"
	}
	return scheme + "://" + host + "/registry"
}

func queryInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return n
}

// ============================================================================
// /api/auth — main demo path: voice → 2-factor verdict → signed event
// ============================================================================

type authResponse struct {
	Event          AuthEventRecord `json:"event"`
	SignedEnvelope *signedEnvelope `json:"signed_envelope"`
}

func (s *server) handleAuth(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	claimedID := strings.TrimSpace(r.FormValue("claimed_official_id"))
	channel := strings.TrimSpace(r.FormValue("channel"))
	if channel == "" {
		channel = "voice"
	}

	file, header, err := r.FormFile("audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "audio required"})
		return
	}
	defer file.Close()
	audioBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	// Parallel: voicebio embedding + Citadel deepfake/transcript scan.
	var (
		embed    *voiceEmbedResult
		embedErr error
		scan     *audioScanResult
		scanErr  error
		wg       sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		embed, embedErr = s.voice.Embed(ctx, audioBytes, header.Filename)
	}()
	go func() {
		defer wg.Done()
		scan, scanErr = s.citadel.ScanAudio(ctx, audioBytes, header.Filename, "secure")
	}()
	wg.Wait()

	rec := AuthEventRecord{
		ID:                newEventID(),
		Channel:           channel,
		SenderClaimed:     claimedID,
		ProcessedAt:       time.Now().UTC(),
		IssuerFingerprint: s.signing.FingerprintShort(),
	}

	if scan != nil {
		dr := scan.DeepfakeRisk
		ir := scan.InjectionRisk
		ur := scan.UltrasonicRsk
		rec.DeepfakeRisk = &dr
		rec.InjectionRisk = &ir
		if ur > 0 {
			rec.UltrasonicRisk = &ur
		}
		rec.Transcript = scan.Transcript
		rec.Signals = scan.Signals
	}

	if embed != nil && claimedID != "" {
		if off, ok := s.enrollments.Get(claimedID); ok {
			rec.OfficialID = off.ID
			rec.OfficialName = off.Name
			sim := CosineSimilarity(off.Embedding, embed.Embedding)
			rec.SpeakerMatch = &sim
		}
	}

	verdict, reason, overall := s.computeVerdict(claimedID, rec, embedErr != nil, scanErr != nil)
	rec.Verdict = verdict
	rec.VerdictReason = reason
	rec.OverallRisk = overall
	rec.LatencyMs = int(time.Since(t0).Milliseconds())

	if embedErr != nil {
		slog.Warn("voicebio failed; verdict best-effort", "err", embedErr)
	}
	if scanErr != nil {
		slog.Warn("citadel failed; verdict best-effort", "err", scanErr)
	}

	env, err := s.signing.Sign(rec)
	if err != nil {
		slog.Warn("sign event envelope", "err", err)
	} else {
		rec.SignedEnvelope = env
	}

	s.events.Add(rec)
	s.sse.Broadcast("auth.event", rec)

	if rec.Verdict == "BLOCKED" {
		s.tryFoundryComment(ctx, rec)
	}

	writeJSON(w, http.StatusOK, authResponse{Event: rec, SignedEnvelope: env})
}

// handleGatewayVerify is the Mighty Morphing Layer 2 path. It keeps the
// passive voice/deepfake checks, then requires an active OOB APPROVE before
// returning VERIFIED.
func (s *server) handleGatewayVerify(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	claimedID := strings.TrimSpace(r.FormValue("claimed_official_id"))
	channel := strings.TrimSpace(r.FormValue("channel"))
	if channel == "" {
		channel = "voice"
	}
	file, header, err := r.FormFile("audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "audio required"})
		return
	}
	defer file.Close()
	audioBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, s.runGatewayVerification(r.Context(), claimedID, channel, audioBytes, header.Filename, t0, ""))
}

func (s *server) handleDemoGatewayVerify(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	kind := strings.ToLower(strings.TrimSpace(r.PathValue("kind")))
	audioPath, filename, err := demoAudioFixture(kind)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	_ = r.ParseMultipartForm(2 << 20)
	claimedID := strings.TrimSpace(r.FormValue("claimed_official_id"))
	if claimedID == "" {
		claimedID = strings.TrimSpace(r.URL.Query().Get("claimed_official_id"))
	}
	if claimedID == "" {
		claimedID = strings.TrimSpace(os.Getenv("MM_DEMO_OFFICIAL_ID"))
	}
	if claimedID == "" {
		claimedID = defaultDemoOfficialID
	}
	channel := strings.TrimSpace(r.FormValue("channel"))
	if channel == "" {
		channel = "demo-" + kind
	}
	audioBytes, err := os.ReadFile(audioPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read demo audio: " + err.Error()})
		return
	}
	// The fixture path is self-contained so the demo can run without a live
	// approval-device tab open: real fixtures auto-APPROVE the simulated OOB,
	// clone fixtures auto-DENY. Passive checks (deepfake + voiceprint) still
	// run end-to-end and can override the simulation by failing earlier.
	simulated := ""
	switch kind {
	case "real":
		simulated = "APPROVE"
	case "clone":
		simulated = "DENY"
	}
	writeJSON(w, http.StatusOK, s.runGatewayVerification(r.Context(), claimedID, channel, audioBytes, filename, t0, simulated))
}

// handleDemoFixtureAudio streams a controlled fixture WAV (real or
// pre-generated Cartesia clone) so the Sign page can demonstrate the layered
// defense without leaving the UI: the fixture is the audio, the user signs
// with their credential, and Verify surfaces voiceprint cosine + audio hash.
func (s *server) handleDemoFixtureAudio(w http.ResponseWriter, r *http.Request) {
	kind := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("kind")))
	path, filename, err := demoAudioFixture(kind)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	audioBytes, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "read fixture: " + err.Error()})
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", `inline; filename="`+filename+`"`)
	w.Header().Set("X-Fixture-Filename", filename)
	w.Header().Set("X-Fixture-Kind", kind)
	w.Header().Set("Access-Control-Expose-Headers", "X-Fixture-Filename, X-Fixture-Kind")
	_, _ = w.Write(audioBytes)
}

func (s *server) handleDemoFixtures(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"fixtures":        demoFixtureCatalog(),
		"provider_counts": demoProviderCounts(),
		"generation": map[string]any{
			"script":       "natsec/scripts/clone_voice.py",
			"source_audio": "natsec/fixtures/user/real_01.wav",
			"providers":    []string{"cartesia", "elevenlabs"},
			"requires":     []string{"explicit consent for the source voice", "CARTESIA_API_KEY and/or ELEVENLABS_API_KEY"},
		},
		"policy": "Backend exposes controlled detector fixtures and a consented generation script. It does not clone public figures or call a cloning provider from the UI.",
	})
}

func (s *server) handleDemoCloneEval(w http.ResponseWriter, r *http.Request) {
	for _, candidate := range []string{
		"../docs/clone_defense_eval_live.json",
		"docs/clone_defense_eval_live.json",
		"natsec/docs/clone_defense_eval_live.json",
	} {
		raw, err := os.ReadFile(candidate)
		if err == nil {
			var body any
			if json.Unmarshal(raw, &body) == nil {
				writeJSON(w, http.StatusOK, body)
				return
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"summary": map[string]any{},
		"rows":    []any{},
		"status":  "not_run",
		"hint":    "Run natsec/scripts/evaluate_clone_defense.py or the local evaluation command to populate natsec/docs/clone_defense_eval_live.json.",
	})
}

type demoFixtureInfo struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Realness    string `json:"realness"`
	Source      string `json:"source"`
	Filename    string `json:"filename"`
	Path        string `json:"path,omitempty"`
	Available   bool   `json:"available"`
	Endpoint    string `json:"endpoint"`
	Description string `json:"description"`
}

func demoFixtureCatalog() []demoFixtureInfo {
	out := make([]demoFixtureInfo, 0, 2)
	for _, item := range []struct {
		kind        string
		label       string
		realness    string
		source      string
		description string
	}{
		{"real", "Controlled enrolled-speaker recording", "real", "local-fixture", "Baseline user-owned recording that v16 currently allows in secure mode."},
		{"clone", "Controlled Cartesia hard-negative", "synthetic", "local-fixture", "Pre-existing Cartesia clone fixture that v16 currently blocks in secure mode; no cloning is performed by this backend."},
	} {
		path, filename, _ := demoAudioFixture(item.kind)
		available := false
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				available = true
			}
		}
		out = append(out, demoFixtureInfo{
			Kind:        item.kind,
			Label:       item.label,
			Realness:    item.realness,
			Source:      item.source,
			Filename:    filename,
			Path:        path,
			Available:   available,
			Endpoint:    "/demo/gateway/" + item.kind,
			Description: item.description,
		})
	}
	return out
}

func demoProviderCounts() map[string]int {
	counts := map[string]int{
		"real":       0,
		"cartesia":   0,
		"elevenlabs": 0,
	}
	for _, dir := range []string{"../fixtures/user", "fixtures/user", "natsec/fixtures/user"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			switch {
			case strings.HasPrefix(name, "real_") && strings.HasSuffix(name, ".wav"):
				counts["real"]++
			case strings.HasPrefix(name, "clone_cartesia_"):
				counts["cartesia"]++
			case strings.HasPrefix(name, "clone_elevenlabs_"):
				counts["elevenlabs"]++
			}
		}
		break
	}
	for _, dir := range []string{"../../tests/artifacts/audio/deepfake_elevenlabs", "../tests/artifacts/audio/deepfake_elevenlabs", "tests/artifacts/audio/deepfake_elevenlabs"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".wav") || strings.HasSuffix(entry.Name(), ".mp3") {
				counts["elevenlabs"]++
			}
		}
		break
	}
	return counts
}

func demoAudioFixture(kind string) (path string, filename string, err error) {
	switch kind {
	case "real":
		if p := strings.TrimSpace(os.Getenv("MM_DEMO_REAL_AUDIO")); p != "" {
			return p, pathBase(p, "real_03.wav"), nil
		}
		filename = "real_03.wav"
	case "clone":
		if p := strings.TrimSpace(os.Getenv("MM_DEMO_CLONE_AUDIO")); p != "" {
			return p, pathBase(p, "clone_cartesia_10.wav"), nil
		}
		filename = "clone_cartesia_10.wav"
	default:
		return "", "", fmt.Errorf("unknown demo audio kind %q", kind)
	}
	for _, candidate := range []string{
		"../fixtures/user/" + filename,
		"fixtures/user/" + filename,
		"natsec/fixtures/user/" + filename,
	} {
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, filename, nil
		}
	}
	return "../fixtures/user/" + filename, filename, nil
}

func pathBase(path string, fallback string) string {
	base := strings.TrimSpace(filepath.Base(path))
	if base == "." || base == "/" || base == "" {
		return fallback
	}
	return base
}

func (s *server) runGatewayVerification(requestCtx context.Context, claimedID, channel string, audioBytes []byte, filename string, t0 time.Time, simulatedOOB string) authResponse {
	ctx, cancel := context.WithTimeout(requestCtx, 30*time.Second)
	defer cancel()

	var (
		embed    *voiceEmbedResult
		embedErr error
		scan     *audioScanResult
		scanErr  error
		wg       sync.WaitGroup
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		embed, embedErr = s.voice.Embed(ctx, audioBytes, filename)
	}()
	go func() {
		defer wg.Done()
		scan, scanErr = s.citadel.ScanAudio(ctx, audioBytes, filename, "secure")
	}()
	wg.Wait()

	rec := AuthEventRecord{
		ID:                newEventID(),
		Channel:           channel,
		SenderClaimed:     claimedID,
		ProcessedAt:       time.Now().UTC(),
		IssuerFingerprint: s.signing.FingerprintShort(),
	}
	if scan != nil {
		dr := scan.DeepfakeRisk
		ir := scan.InjectionRisk
		ur := scan.UltrasonicRsk
		rec.DeepfakeRisk = &dr
		rec.InjectionRisk = &ir
		if ur > 0 {
			rec.UltrasonicRisk = &ur
		}
		rec.Transcript = scan.Transcript
		rec.Signals = scan.Signals
	}
	if embed != nil && claimedID != "" {
		if off, ok := s.enrollments.Get(claimedID); ok {
			rec.OfficialID = off.ID
			rec.OfficialName = off.Name
			sim := CosineSimilarity(off.Embedding, embed.Embedding)
			rec.SpeakerMatch = &sim
		}
	}

	passiveVerdict, passiveReason, overall := s.computeVerdict(claimedID, rec, embedErr != nil, scanErr != nil)
	rec.OverallRisk = overall
	if scanErr != nil {
		rec.Verdict = "BLOCKED"
		rec.VerdictReason = "citadel-unavailable: " + scanErr.Error()
		rec.OOBResponse = "SKIPPED"
		rec.OverallRisk = max(rec.OverallRisk, 100)
	} else if passiveVerdict != "VERIFIED" {
		rec.Verdict = passiveVerdict
		rec.VerdictReason = passiveReason
		rec.OOBResponse = "SKIPPED"
	} else if simulatedOOB != "" {
		rec.OOBResponse = simulatedOOB + "-SIMULATED"
		switch strings.ToUpper(simulatedOOB) {
		case "APPROVE":
			rec.Verdict = "VERIFIED"
			rec.VerdictReason = "three-factor pass (OOB simulated for demo fixture path)"
		case "DENY":
			rec.Verdict = "BLOCKED"
			rec.VerdictReason = "simulated OOB deny (clone-fixture path)"
			rec.OverallRisk = max(rec.OverallRisk, 100)
		default:
			rec.Verdict = "BLOCKED"
			rec.VerdictReason = "unknown simulated OOB verdict"
			rec.OverallRisk = max(rec.OverallRisk, 100)
		}
	} else if s.oob == nil {
		rec.Verdict = "BLOCKED"
		rec.VerdictReason = "OOB-unavailable"
		rec.OOBResponse = "TIMEOUT"
		rec.OverallRisk = max(rec.OverallRisk, 100)
	} else {
		oob := s.oob.Prompt(ctx, claimedID, rec.Transcript, rec.Signals)
		rec.OOBResponse = oob.Verdict
		rec.OOBPromptID = oob.PromptID
		switch oob.Verdict {
		case "APPROVE":
			rec.Verdict = "VERIFIED"
			rec.VerdictReason = "three-factor pass"
		case "DENY":
			rec.Verdict = "BLOCKED"
			rec.VerdictReason = "official denied OOB"
			rec.OverallRisk = max(rec.OverallRisk, 100)
		default:
			rec.Verdict = "BLOCKED"
			rec.VerdictReason = oob.Reason
			if rec.VerdictReason == "" {
				rec.VerdictReason = "OOB-timeout"
			}
			rec.OverallRisk = max(rec.OverallRisk, 100)
		}
	}
	rec.LatencyMs = int(time.Since(t0).Milliseconds())

	if embedErr != nil {
		slog.Warn("voicebio failed in gateway verify", "err", embedErr)
	}
	if scanErr != nil {
		slog.Warn("citadel failed in gateway verify", "err", scanErr)
	}
	env, err := s.signing.Sign(rec)
	if err != nil {
		slog.Warn("sign gateway event envelope", "err", err)
	} else {
		rec.SignedEnvelope = env
	}
	s.events.Add(rec)
	s.sse.Broadcast("auth.event", rec)
	if s.audit != nil {
		if _, err := s.audit.Append("gateway.verified", claimedID, "", map[string]any{
			"event_id":       rec.ID,
			"verdict":        rec.Verdict,
			"reason":         rec.VerdictReason,
			"speaker_match":  rec.SpeakerMatch,
			"deepfake_risk":  rec.DeepfakeRisk,
			"oob_response":   rec.OOBResponse,
			"oob_prompt_id":  rec.OOBPromptID,
			"overall_risk":   rec.OverallRisk,
			"claimed_sender": claimedID,
		}); err != nil {
			slog.Warn("append audit gateway verify", "err", err)
		}
	}
	if rec.Verdict == "BLOCKED" {
		s.tryFoundryComment(ctx, rec)
	}
	return authResponse{Event: rec, SignedEnvelope: env}
}

// computeVerdict implements the two-factor decision matrix.
func (s *server) computeVerdict(claimedID string, rec AuthEventRecord, voiceErr, scanErr bool) (verdict, reason string, overallRisk int) {
	dfThreshold := s.cfg.DeepfakeBlockRisk
	injThreshold := s.cfg.InjectionBlockRisk
	matchThreshold := s.cfg.SpeakerMatchThreshold

	// Hard service-failure guards.
	if voiceErr && scanErr {
		return "BLOCKED", "fail-closed: both voice biometric and deepfake services unreachable", 100
	}

	df := 0
	if rec.DeepfakeRisk != nil {
		df = *rec.DeepfakeRisk
	}
	inj := 0
	if rec.InjectionRisk != nil {
		inj = *rec.InjectionRisk
	}
	overallRisk = max3(df, inj, ultrasonicVal(rec.UltrasonicRisk))

	// Revocation check.
	if claimedID != "" {
		if off, ok := s.enrollments.Get(claimedID); ok && off.Status == "revoked" {
			return "BLOCKED", "claimed identity revoked", max(overallRisk, 100)
		}
	}

	// Hard injection or ultrasonic — block regardless of speaker match.
	if inj >= injThreshold {
		return "BLOCKED", fmt.Sprintf("spoken prompt-injection risk %d ≥ %d", inj, injThreshold), overallRisk
	}
	if rec.UltrasonicRisk != nil && *rec.UltrasonicRisk >= 70 {
		return "BLOCKED", fmt.Sprintf("ultrasonic carrier risk %d", *rec.UltrasonicRisk), overallRisk
	}

	// Deepfake check.
	deepfakeFail := df >= dfThreshold
	speakerFail := false
	hasSpeakerScore := rec.SpeakerMatch != nil
	if rec.SpeakerMatch != nil {
		speakerFail = *rec.SpeakerMatch < matchThreshold
	}

	switch {
	case claimedID == "" && deepfakeFail:
		return "BLOCKED", fmt.Sprintf("synthetic audio risk %d ≥ %d", df, dfThreshold), overallRisk
	case claimedID == "" && !deepfakeFail:
		return "WARN", "no claimed identity; passive monitor only", overallRisk
	case claimedID != "" && !hasSpeakerScore:
		return "BLOCKED", "claimed identity could not be matched to a speaker enrollment", max(overallRisk, 100)
	case deepfakeFail && !speakerFail:
		return "BLOCKED", fmt.Sprintf("high-fidelity clone of enrolled official (deepfake=%d, speaker=%.2f)", df, *rec.SpeakerMatch), overallRisk
	case !deepfakeFail && speakerFail:
		return "BLOCKED", fmt.Sprintf("real audio but wrong person (deepfake=%d, speaker=%.2f vs threshold %.2f)", df, *rec.SpeakerMatch, matchThreshold), overallRisk
	case deepfakeFail && speakerFail:
		return "BLOCKED", fmt.Sprintf("synthetic audio of unknown speaker (deepfake=%d, speaker=%.2f)", df, *rec.SpeakerMatch), overallRisk
	default:
		return "VERIFIED", "two-factor pass", overallRisk
	}
}

// ============================================================================
// /api/scan/* — AI-agent input/output guardrails (text, image, document, output)
// ============================================================================

type scanRequest struct {
	Input string `json:"input"`
	Mode  string `json:"mode,omitempty"`
}

func (s *server) handleScanText(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	var req scanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.Input) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "input required"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	scan, err := s.citadel.ScanText(ctx, req.Input, req.Mode)
	rec := s.recordFromTextScan("text", scan, err)
	rec.LatencyMs = int(time.Since(t0).Milliseconds())
	env, _ := s.signing.Sign(rec)
	rec.SignedEnvelope = env
	s.events.Add(rec)
	s.sse.Broadcast("auth.event", rec)
	writeJSON(w, http.StatusOK, authResponse{Event: rec, SignedEnvelope: env})
}

func (s *server) handleScanImage(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "image required"})
		return
	}
	defer file.Close()
	imgBytes, err := io.ReadAll(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	scan, err := s.citadel.ScanImage(ctx, imgBytes, header.Filename, "secure")
	rec := s.recordFromTextScan("image", scan, err)
	rec.LatencyMs = int(time.Since(t0).Milliseconds())
	env, _ := s.signing.Sign(rec)
	rec.SignedEnvelope = env
	s.events.Add(rec)
	s.sse.Broadcast("auth.event", rec)
	writeJSON(w, http.StatusOK, authResponse{Event: rec, SignedEnvelope: env})
}

func (s *server) handleScanDocument(w http.ResponseWriter, r *http.Request) {
	// Same plumbing as image; Citadel routes PDFs through vision pipeline.
	s.handleScanImage(w, r)
}

func (s *server) handleScanOutput(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	var req scanRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	// For outbound scanning we use the same text gateway; Citadel detects
	// secrets, classified markers, exfil patterns, etc.
	scan, err := s.citadel.ScanText(ctx, req.Input, "secure")
	rec := s.recordFromTextScan("output", scan, err)
	rec.LatencyMs = int(time.Since(t0).Milliseconds())
	env, _ := s.signing.Sign(rec)
	rec.SignedEnvelope = env
	s.events.Add(rec)
	s.sse.Broadcast("auth.event", rec)
	writeJSON(w, http.StatusOK, authResponse{Event: rec, SignedEnvelope: env})
}

func (s *server) recordFromTextScan(channel string, scan *textScanResult, scanErr error) AuthEventRecord {
	rec := AuthEventRecord{
		ID:                newEventID(),
		Channel:           channel,
		ProcessedAt:       time.Now().UTC(),
		IssuerFingerprint: s.signing.FingerprintShort(),
	}
	if scanErr != nil || scan == nil {
		rec.Verdict = "WARN"
		rec.VerdictReason = "scan failed: " + safeErr(scanErr)
		rec.OverallRisk = 0
		return rec
	}
	rec.OverallRisk = scan.Risk
	rec.Signals = scan.Signals
	if scan.InjectionRisk > 0 {
		ir := scan.InjectionRisk
		rec.InjectionRisk = &ir
	}
	switch strings.ToUpper(scan.Action) {
	case "BLOCK", "BLOCKED":
		rec.Verdict = "BLOCKED"
		rec.VerdictReason = fmt.Sprintf("citadel %s risk=%d", strings.Join(scan.Categories, ","), scan.Risk)
	case "WARN":
		rec.Verdict = "WARN"
		rec.VerdictReason = fmt.Sprintf("citadel WARN risk=%d", scan.Risk)
	default:
		rec.Verdict = "VERIFIED"
		rec.VerdictReason = "citadel ALLOW"
	}
	return rec
}

// ============================================================================
// /verify/* — offline-verifiable signed events + pubkey
// ============================================================================

func (s *server) handleVerifyEvent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rec, ok := s.events.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "event not found"})
		return
	}
	writeJSON(w, http.StatusOK, rec)
}

func (s *server) handleVerifyPubkey(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"alg":         "ed25519",
		"public_key":  s.signing.PublicKeyHex(),
		"fingerprint": s.signing.FingerprintShort(),
	})
}

// ============================================================================
// /api/events — SSE stream
// ============================================================================

func (s *server) handleEventsSSE(w http.ResponseWriter, r *http.Request) {
	s.sse.ServeHTTP(w, r)
}

func (s *server) handleRecentEvents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": s.events.Recent(50),
	})
}

// ============================================================================
// Foundry write-through (best effort)
// ============================================================================

// tryFoundryComment attempts to write a RouteAlertComment to Foundry on BLOCK.
// Best effort: if the action signature doesn't match what's installed in this
// stack, we log and move on. The local signed audit trail is the source of
// truth.
func (s *server) tryFoundryComment(ctx context.Context, rec AuthEventRecord) {
	if !s.foundry.Configured() {
		return
	}
	// We can't create RouteAlerts directly (no createObject action found in
	// the current ontology probe). Best alternative for the demo: log to
	// stdout and rely on local signed audit. A Sunday-morning enhancement
	// would map to whatever existing alert workflow exists in the ontology.
	slog.Info("foundry write skipped (no compatible create-alert action installed); local audit retained",
		"event_id", rec.ID, "verdict", rec.Verdict)
}

// ============================================================================
// Helpers
// ============================================================================

func decodeJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	return decodeStrict(body, v)
}

func decodeStrict(body []byte, v any) error {
	return jsonUnmarshal(body, v)
}

// jsonUnmarshal is wrapped so we can swap in a stricter decoder later if
// needed (DisallowUnknownFields, etc.).
func jsonUnmarshal(body []byte, v any) error {
	return jsonDecodeStrict(body, v)
}

func jsonDecodeStrict(body []byte, v any) error {
	dec := newJSONDecoder(body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func newEventID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	return "evt_" + hex.EncodeToString(b[:])
}

func safeErr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func max3(a, b, c int) int {
	m := a
	if b > m {
		m = b
	}
	if c > m {
		m = c
	}
	return m
}

func ultrasonicVal(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
