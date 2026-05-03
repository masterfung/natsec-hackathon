// Ed25519 server identity. Every verdict and enrollment record we emit is signed
// by this key. The public key is published at /verify/pubkey so any third party
// can verify the audit trail offline (the basis for the QR-code provenance UX).
//
// In production this would be HSM-backed; for the hackathon prototype we
// generate a fresh key on first run and persist it to disk in .secrets/.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

type signingKey struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	id   string // sha256 of pub, hex; durable identifier for this server
}

// loadOrCreateSigningKey tries to read an existing key from path; if absent,
// generates and persists a new Ed25519 key.
func loadOrCreateSigningKey(path string) (*signingKey, error) {
	if path == "" {
		path = ".secrets/server-ed25519.key"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("mkdir secrets: %w", err)
	}

	if data, err := os.ReadFile(path); err == nil && len(data) >= ed25519.PrivateKeySize {
		priv := ed25519.PrivateKey(data[:ed25519.PrivateKeySize])
		return makeSigningKey(priv), nil
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	if err := os.WriteFile(path, priv, 0o600); err != nil {
		return nil, fmt.Errorf("persist key: %w", err)
	}
	_ = pub // pub embedded in priv
	return makeSigningKey(priv), nil
}

func loadOrCreateSecretBytes(path string, n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("invalid secret size %d", n)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("mkdir secrets: %w", err)
	}
	if data, err := os.ReadFile(path); err == nil && len(data) == n {
		return data, nil
	}
	secret := make([]byte, n)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	if err := os.WriteFile(path, secret, 0o600); err != nil {
		return nil, fmt.Errorf("persist secret: %w", err)
	}
	return secret, nil
}

func makeSigningKey(priv ed25519.PrivateKey) *signingKey {
	pub := priv.Public().(ed25519.PublicKey)
	hash := sha256.Sum256(pub)
	return &signingKey{priv: priv, pub: pub, id: hex.EncodeToString(hash[:])}
}

// PublicKeyHex returns the hex-encoded public key (32 bytes -> 64 hex chars).
func (s *signingKey) PublicKeyHex() string {
	return hex.EncodeToString(s.pub)
}

// FingerprintShort returns a short fingerprint (first 12 hex chars of sha256(pub))
// suitable for UI display.
func (s *signingKey) FingerprintShort() string {
	return s.id[:12]
}

// signedEnvelope wraps any JSON-marshalable payload with a signature so a third
// party with the public key can verify the payload was produced by this server
// without having to call back.
type signedEnvelope struct {
	V         string          `json:"v"`         // schema version
	Issuer    string          `json:"issuer"`    // public key fingerprint
	IssuedAt  string          `json:"issued_at"` // RFC3339
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"` // base64(ed25519(canonical(envelope without signature)))
}

// Sign produces a signed envelope around `payload`. Marshals deterministically.
func (s *signingKey) Sign(payload any) (*signedEnvelope, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}
	env := &signedEnvelope{
		V:        "mm.envelope.v1",
		Issuer:   s.FingerprintShort(),
		IssuedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Payload:  body,
	}
	signedBytes, err := canonicalForSigning(env)
	if err != nil {
		return nil, err
	}
	sig := ed25519.Sign(s.priv, signedBytes)
	env.Signature = base64.StdEncoding.EncodeToString(sig)
	return env, nil
}

// Verify takes an envelope and checks the signature against the server's public
// key. For external offline verification, the consumer would use the published
// public key at /verify/pubkey.
func (s *signingKey) Verify(env *signedEnvelope) error {
	if env == nil || env.Signature == "" {
		return errors.New("nil or unsigned envelope")
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return fmt.Errorf("decode sig: %w", err)
	}
	body, err := canonicalForSigning(env)
	if err != nil {
		return err
	}
	if !ed25519.Verify(s.pub, body, sig) {
		return errors.New("signature verification failed")
	}
	return nil
}

// canonicalForSigning emits the bytes to-be-signed: the envelope serialized
// with the Signature field empty. We use Go's stable map ordering on structs
// (alphabetical struct field order is deterministic) and a single JSON encode
// to keep this consistent on both sides.
func canonicalForSigning(env *signedEnvelope) ([]byte, error) {
	clone := *env
	clone.Signature = ""
	return json.Marshal(&clone)
}

// HashEmbeddingFingerprint returns a short, human-readable fingerprint for an
// embedding (truncated SHA-256 of float32 LE bytes). Used as a "voiceprint
// fingerprint" badge in the SOC console — it's not the embedding itself, it's
// a tamper-evident hash for audit display.
func HashEmbeddingFingerprint(emb []float32) string {
	h := sha256.New()
	buf := make([]byte, 4)
	for _, v := range emb {
		binary.LittleEndian.PutUint32(buf, math.Float32bits(v))
		h.Write(buf)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}
