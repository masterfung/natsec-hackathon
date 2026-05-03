package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	identityEnvelopeVersion = "mm.identity.v1"
	mediaSignatureVersion   = "mm.media.sig.v1"
)

// identityEnvelope is the user's signing credential. It contains the private
// key and is delivered ONCE on enrollment in the `enrollment_secret` block.
// Subsequent registry reads only ever return publicIdentityEnvelope.
type identityEnvelope struct {
	V           string `json:"v"`
	Alg         string `json:"alg"`
	ID          string `json:"id"`
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"`
	RegistryURL string `json:"registry_url"`
	Fingerprint string `json:"fingerprint"`
	IssuedAt    string `json:"issued_at"`
}

// publicCredential is what the user keeps after enrollment. The signing key
// is in `Secret`; the public envelope mirrors what is stored in the registry
// and is safe to share.
type publicCredential struct {
	Identity *publicIdentityEnvelope `json:"identity_envelope"`
	Secret   *enrollmentSecret       `json:"enrollment_secret"`
}

// enrollmentSecret is the one-time delivery of the user's private signing key.
// It is returned by /api/enroll and /registry/enroll and is never persisted in
// any registry response. Treat the file containing this object the way you
// would treat an SSH private key.
type enrollmentSecret struct {
	Warning    string `json:"warning"`
	ID         string `json:"id"`
	Alg        string `json:"alg"`
	PrivateKey string `json:"private_key"`
	IssuedAt   string `json:"issued_at"`
	// Roadmap fields for honest forward-looking story on the demo screen.
	PostQuantumRoadmap string `json:"post_quantum_roadmap,omitempty"`
	PasskeyRoadmap     string `json:"passkey_roadmap,omitempty"`
}

const enrollmentSecretWarning = "PRIVATE SIGNING KEY — save this file now and never share it. " +
	"Anyone with this key can sign media as you. The registry only stores your public key."

const postQuantumRoadmapNote = "Demo signs with Ed25519. Production roadmap: ML-DSA-65 (FIPS 204) " +
	"hybrid Ed25519+ML-DSA so signatures stay verifiable after a CRQC milestone."

const passkeyRoadmapNote = "Demo issues an Ed25519 keypair server-side. Production roadmap: " +
	"WebAuthn passkeys with the private key sealed in Secure Enclave / TPM so it cannot be exfiltrated."

// publicIdentityEnvelope is the public-only view of an identityEnvelope. It is
// the only form returned by registry lookups and is the form embedded in the
// enroll response alongside enrollmentSecret.
type publicIdentityEnvelope struct {
	V           string `json:"v"`
	Alg         string `json:"alg"`
	ID          string `json:"id"`
	PublicKey   string `json:"public_key"`
	RegistryURL string `json:"registry_url"`
	Fingerprint string `json:"fingerprint"`
	IssuedAt    string `json:"issued_at"`
}

func (e *identityEnvelope) Public() *publicIdentityEnvelope {
	if e == nil {
		return nil
	}
	return &publicIdentityEnvelope{
		V:           e.V,
		Alg:         e.Alg,
		ID:          e.ID,
		PublicKey:   e.PublicKey,
		RegistryURL: e.RegistryURL,
		Fingerprint: e.Fingerprint,
		IssuedAt:    e.IssuedAt,
	}
}

func (e *identityEnvelope) Secret() *enrollmentSecret {
	if e == nil {
		return nil
	}
	return &enrollmentSecret{
		Warning:            enrollmentSecretWarning,
		ID:                 e.ID,
		Alg:                e.Alg,
		PrivateKey:         e.PrivateKey,
		IssuedAt:           e.IssuedAt,
		PostQuantumRoadmap: postQuantumRoadmapNote,
		PasskeyRoadmap:     passkeyRoadmapNote,
	}
}

func newIdentityEnvelope(id, registryURL string) (*identityEnvelope, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate user keypair: %w", err)
	}
	pubHex := hex.EncodeToString(pub)
	return &identityEnvelope{
		V:           identityEnvelopeVersion,
		Alg:         "ed25519",
		ID:          id,
		PublicKey:   pubHex,
		PrivateKey:  hex.EncodeToString(priv),
		RegistryURL: registryURL,
		Fingerprint: shortSHA512Hex([]byte(pubHex)),
		IssuedAt:    time.Now().UTC().Format(time.RFC3339Nano),
	}, nil
}

// parseIdentityEnvelope accepts either the new credential-bundle format
// (publicCredential = {identity_envelope, enrollment_secret}) or the legacy
// flat identityEnvelope with private_key inline. The bundle form is what the
// enroll endpoint now writes; the legacy form is kept for credentials saved
// before this change.
func parseIdentityEnvelope(raw string) (*identityEnvelope, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("identity envelope is empty")
	}

	// Try bundle form first.
	var bundle struct {
		Identity *publicIdentityEnvelope `json:"identity_envelope"`
		Secret   *enrollmentSecret       `json:"enrollment_secret"`
	}
	if err := json.Unmarshal([]byte(raw), &bundle); err == nil &&
		bundle.Identity != nil && bundle.Secret != nil &&
		bundle.Identity.PublicKey != "" && bundle.Secret.PrivateKey != "" {
		if bundle.Identity.ID != "" && bundle.Secret.ID != "" && bundle.Identity.ID != bundle.Secret.ID {
			return nil, errors.New("credential bundle id mismatch between identity_envelope and enrollment_secret")
		}
		id := bundle.Identity.ID
		if id == "" {
			id = bundle.Secret.ID
		}
		alg := bundle.Identity.Alg
		if alg == "" {
			alg = bundle.Secret.Alg
		}
		if alg == "" {
			alg = "ed25519"
		}
		return &identityEnvelope{
			V:           orDefault(bundle.Identity.V, identityEnvelopeVersion),
			Alg:         alg,
			ID:          id,
			PublicKey:   bundle.Identity.PublicKey,
			PrivateKey:  bundle.Secret.PrivateKey,
			RegistryURL: bundle.Identity.RegistryURL,
			Fingerprint: bundle.Identity.Fingerprint,
			IssuedAt:    orDefault(bundle.Identity.IssuedAt, bundle.Secret.IssuedAt),
		}, nil
	}

	// Legacy flat form.
	var env identityEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return nil, fmt.Errorf("decode identity envelope: %w", err)
	}
	if env.ID == "" || env.PublicKey == "" || env.PrivateKey == "" {
		return nil, errors.New("identity envelope missing id/public_key/private_key (or supply identity_envelope + enrollment_secret bundle)")
	}
	if env.V == "" {
		env.V = identityEnvelopeVersion
	}
	if env.Alg == "" {
		env.Alg = "ed25519"
	}
	return &env, nil
}

func orDefault(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

type mediaSignatureEnvelope struct {
	V                   string `json:"v"`
	Alg                 string `json:"alg"`
	IssuerID            string `json:"issuer_id"`
	Fingerprint         string `json:"fingerprint"`
	Timestamp           string `json:"timestamp"`
	Purpose             string `json:"purpose"`
	AudioSHA512         string `json:"audio_sha512"`
	CanonicalPayloadB64 string `json:"canonical_payload_b64"`
	Signature           string `json:"signature"`
}

type mediaSigningPayload struct {
	AudioSHA512 string `json:"audio_sha512"`
	Timestamp   string `json:"timestamp"`
	Purpose     string `json:"purpose"`
}

func signMediaEnvelope(issuerID, fingerprint, purpose string, audioBytes []byte, privateKeyHex string) (*mediaSignatureEnvelope, error) {
	priv, err := decodeEd25519PrivateKey(privateKeyHex)
	if err != nil {
		return nil, err
	}
	if purpose == "" {
		purpose = "mighty-morphing.media"
	}
	ts := time.Now().UTC().Format(time.RFC3339Nano)
	audioHash := sha512Hex(audioBytes)
	payload := mediaSigningPayload{
		AudioSHA512: audioHash,
		Timestamp:   ts,
		Purpose:     purpose,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(priv, payloadBytes)
	return &mediaSignatureEnvelope{
		V:                   mediaSignatureVersion,
		Alg:                 "ed25519",
		IssuerID:            issuerID,
		Fingerprint:         fingerprint,
		Timestamp:           ts,
		Purpose:             purpose,
		AudioSHA512:         audioHash,
		CanonicalPayloadB64: base64.StdEncoding.EncodeToString(payloadBytes),
		Signature:           base64.StdEncoding.EncodeToString(sig),
	}, nil
}

func verifyMediaEnvelope(env *mediaSignatureEnvelope, audioBytes []byte, publicKeyHex string) (hashMatches bool, signatureValid bool, err error) {
	if env == nil {
		return false, false, errors.New("nil media signature envelope")
	}
	pub, err := decodeEd25519PublicKey(publicKeyHex)
	if err != nil {
		return false, false, err
	}
	hashMatches = strings.EqualFold(env.AudioSHA512, sha512Hex(audioBytes))
	payload := mediaSigningPayload{
		AudioSHA512: env.AudioSHA512,
		Timestamp:   env.Timestamp,
		Purpose:     env.Purpose,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return hashMatches, false, err
	}
	if env.CanonicalPayloadB64 != "" {
		decoded, decodeErr := base64.StdEncoding.DecodeString(env.CanonicalPayloadB64)
		if decodeErr != nil {
			return hashMatches, false, fmt.Errorf("decode canonical payload: %w", decodeErr)
		}
		if string(decoded) != string(payloadBytes) {
			return hashMatches, false, errors.New("signature envelope canonical payload mismatch")
		}
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return hashMatches, false, fmt.Errorf("decode media signature: %w", err)
	}
	signatureValid = ed25519.Verify(pub, payloadBytes, sig)
	return hashMatches, signatureValid, nil
}

func publicKeyFromPrivateHex(privateKeyHex string) (string, error) {
	priv, err := decodeEd25519PrivateKey(privateKeyHex)
	if err != nil {
		return "", err
	}
	pub := priv.Public().(ed25519.PublicKey)
	return hex.EncodeToString(pub), nil
}

func decodeEd25519PrivateKey(raw string) (ed25519.PrivateKey, error) {
	b, err := hex.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(b) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid Ed25519 private key length %d", len(b))
	}
	return ed25519.PrivateKey(b), nil
}

func decodeEd25519PublicKey(raw string) (ed25519.PublicKey, error) {
	b, err := hex.DecodeString(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key length %d", len(b))
	}
	return ed25519.PublicKey(b), nil
}

func sha512Hex(b []byte) string {
	sum := sha512.Sum512(b)
	return hex.EncodeToString(sum[:])
}

func shortSHA512Hex(b []byte) string {
	return sha512Hex(b)[:16]
}

func HashEmbeddingSHA512(emb []float32, salt []byte) string {
	h := sha512.New()
	if len(salt) > 0 {
		h.Write(salt)
	}
	buf := make([]byte, 4)
	for _, v := range emb {
		binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
		h.Write(buf)
	}
	return hex.EncodeToString(h.Sum(nil))
}
