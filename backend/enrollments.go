// Local enrollment store. Persists official identities + voice embeddings to
// data/enrollments.json. In a production deployment the embedding would live
// in an HSM or encrypted enclave; for the hackathon we keep it on disk and
// gitignore it.
//
// Each record carries a SHA-512 voiceprint fingerprint for public registry
// display and an Ed25519 signature over the canonical record bytes for audit
// trail. Rotation (revocation + re-enrollment) is supported.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// Official is the persisted record. Embedding is kept locally for matching;
// the public-facing JSON in /api/officials redacts it.
type Official struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Role                 string     `json:"role,omitempty"`
	PhotoURI             string     `json:"photo_uri,omitempty"`
	EnrollmentCenter     string     `json:"enrollment_center,omitempty"`
	EnrollmentDate       time.Time  `json:"enrollment_date"`
	PublicKey            string     `json:"public_key,omitempty"`
	VoiceprintSHA512     string     `json:"sha512_fingerprint,omitempty"`
	EmbeddingDim         int        `json:"embedding_dim"`
	EmbeddingFingerprint string     `json:"embedding_fingerprint"` // truncated sha256 of emb LE bytes
	EmbeddingQuality     float64    `json:"embedding_quality"`
	SampleDurationS      float64    `json:"sample_duration_s"`
	Status               string     `json:"status"` // active | revoked
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
	IssuerFingerprint    string     `json:"issuer_fingerprint"` // server signing key fingerprint
	Signature            string     `json:"signature"`          // base64 Ed25519 over (record without Signature)
	Embedding            []float32  `json:"embedding"`          // raw — local file only; redacted in API
	// Passkeys is the WebAuthn credential set bound to this official. The
	// private keys live in the user's authenticator (Secure Enclave / TPM /
	// hardware token) and never reach this server. We only persist credential
	// ID, public key, AAGUID, sign counter, and flags.
	Passkeys []webauthn.Credential `json:"passkeys,omitempty"`
}

// PublicView is what we return to clients (no embedding, no signature noise).
type OfficialPublic struct {
	ID                   string     `json:"id"`
	Name                 string     `json:"name"`
	Role                 string     `json:"role,omitempty"`
	PhotoURI             string     `json:"photo_uri,omitempty"`
	EnrollmentCenter     string     `json:"enrollment_center,omitempty"`
	EnrollmentDate       time.Time  `json:"enrollment_date"`
	PublicKey            string     `json:"public_key,omitempty"`
	VoiceprintSHA512     string     `json:"sha512_fingerprint,omitempty"`
	EmbeddingDim         int        `json:"embedding_dim"`
	EmbeddingFingerprint string     `json:"embedding_fingerprint"`
	EmbeddingQuality     float64    `json:"embedding_quality"`
	SampleDurationS      float64    `json:"sample_duration_s"`
	Status               string     `json:"status"`
	RevokedAt            *time.Time `json:"revoked_at,omitempty"`
	IssuerFingerprint    string     `json:"issuer_fingerprint"`
}

type OfficialRegistryPublic struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Role             string     `json:"role,omitempty"`
	VoiceprintSHA512 string     `json:"sha512_fingerprint"`
	PublicKey        string     `json:"public_key"`
	Status           string     `json:"status"`
	RegistryStatus   string     `json:"registry_status"`
	Source           string     `json:"source"`
	Realness         string     `json:"realness"`
	EnrolledAt       time.Time  `json:"enrolled_at"`
	EnrollmentCenter string     `json:"enrollment_center,omitempty"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
}

func (o *Official) Public() OfficialPublic {
	return OfficialPublic{
		ID:                   o.ID,
		Name:                 o.Name,
		Role:                 o.Role,
		PhotoURI:             o.PhotoURI,
		EnrollmentCenter:     o.EnrollmentCenter,
		EnrollmentDate:       o.EnrollmentDate,
		PublicKey:            o.PublicKey,
		VoiceprintSHA512:     o.VoiceprintSHA512,
		EmbeddingDim:         o.EmbeddingDim,
		EmbeddingFingerprint: o.EmbeddingFingerprint,
		EmbeddingQuality:     o.EmbeddingQuality,
		SampleDurationS:      o.SampleDurationS,
		Status:               o.Status,
		RevokedAt:            o.RevokedAt,
		IssuerFingerprint:    o.IssuerFingerprint,
	}
}

func (o *Official) RegistryPublic() OfficialRegistryPublic {
	return OfficialRegistryPublic{
		ID:               o.ID,
		Name:             o.Name,
		Role:             o.Role,
		VoiceprintSHA512: o.VoiceprintSHA512,
		PublicKey:        o.PublicKey,
		Status:           o.Status,
		RegistryStatus:   o.Status,
		Source:           "local-enrollment",
		Realness:         "enrolled-real-speaker",
		EnrolledAt:       o.EnrollmentDate,
		EnrollmentCenter: o.EnrollmentCenter,
		RevokedAt:        o.RevokedAt,
	}
}

type duplicateEnrollmentError struct {
	ID string
}

func (e duplicateEnrollmentError) Error() string {
	return fmt.Sprintf("official id %q is already enrolled and active; revoke it before re-enrolling", e.ID)
}

type enrollmentStore struct {
	path    string
	mu      sync.RWMutex
	by_id   map[string]*Official
	signing *signingKey
	salt    []byte
}

func newEnrollmentStore(path string, sk *signingKey, salt []byte) (*enrollmentStore, error) {
	if path == "" {
		path = "data/enrollments.json"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data: %w", err)
	}
	s := &enrollmentStore{
		path:    path,
		by_id:   make(map[string]*Official),
		signing: sk,
		salt:    salt,
	}
	if err := s.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

// Add (or replace) an enrollment. Sets IssuerFingerprint + Signature.
func (s *enrollmentStore) Add(off *Official) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if off.ID == "" {
		return errors.New("official id required")
	}
	if existing, ok := s.by_id[off.ID]; ok && existing.Status != "revoked" {
		return duplicateEnrollmentError{ID: off.ID}
	}
	if off.Status == "" {
		off.Status = "active"
	}
	if off.EnrollmentDate.IsZero() {
		off.EnrollmentDate = time.Now().UTC()
	}
	off.EmbeddingFingerprint = HashEmbeddingFingerprint(off.Embedding)
	off.VoiceprintSHA512 = HashEmbeddingSHA512(off.Embedding, s.salt)
	off.EmbeddingDim = len(off.Embedding)
	off.IssuerFingerprint = s.signing.FingerprintShort()

	// Sign the record (embedding excluded from canonical signed form for stability;
	// fingerprint binds it.)
	canon := *off
	canon.Signature = ""
	canon.Embedding = nil
	env, err := s.signing.Sign(canon)
	if err != nil {
		return fmt.Errorf("sign enrollment: %w", err)
	}
	off.Signature = env.Signature

	s.by_id[off.ID] = off
	return s.persist()
}

func (s *enrollmentStore) Get(id string) (*Official, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	o, ok := s.by_id[id]
	return o, ok
}

func (s *enrollmentStore) List() []OfficialPublic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]OfficialPublic, 0, len(s.by_id))
	for _, o := range s.by_id {
		out = append(out, o.Public())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EnrollmentDate.After(out[j].EnrollmentDate) })
	return out
}

func (s *enrollmentStore) RegistryList() []OfficialRegistryPublic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]OfficialRegistryPublic, 0, len(s.by_id))
	for _, o := range s.by_id {
		out = append(out, o.RegistryPublic())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EnrolledAt.After(out[j].EnrolledAt) })
	return out
}

func (s *enrollmentStore) RegistrySearch(q, status, source, realness string, limit int) []OfficialRegistryPublic {
	s.mu.RLock()
	defer s.mu.RUnlock()
	q = strings.ToLower(strings.TrimSpace(q))
	status = strings.ToLower(strings.TrimSpace(status))
	source = strings.ToLower(strings.TrimSpace(source))
	realness = strings.ToLower(strings.TrimSpace(realness))
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	out := make([]OfficialRegistryPublic, 0, len(s.by_id))
	for _, o := range s.by_id {
		pub := o.RegistryPublic()
		if status != "" && strings.ToLower(pub.Status) != status && strings.ToLower(pub.RegistryStatus) != status {
			continue
		}
		if source != "" && strings.ToLower(pub.Source) != source {
			continue
		}
		if realness != "" && strings.ToLower(pub.Realness) != realness {
			continue
		}
		if q != "" && !registryRecordMatches(pub, q) {
			continue
		}
		out = append(out, pub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].EnrolledAt.After(out[j].EnrolledAt) })
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func (s *enrollmentStore) RegistryLookup(id, fingerprint, publicKey string) (*OfficialRegistryPublic, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id = strings.TrimSpace(id)
	fingerprint = strings.ToLower(strings.TrimSpace(fingerprint))
	publicKey = strings.ToLower(strings.TrimSpace(publicKey))
	for _, o := range s.by_id {
		pub := o.RegistryPublic()
		switch {
		case id != "" && pub.ID == id:
			return &pub, true
		case fingerprint != "" && strings.ToLower(pub.VoiceprintSHA512) == fingerprint:
			return &pub, true
		case publicKey != "" && strings.ToLower(pub.PublicKey) == publicKey:
			return &pub, true
		}
	}
	return nil, false
}

func registryRecordMatches(pub OfficialRegistryPublic, q string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		pub.ID,
		pub.Name,
		pub.Role,
		pub.VoiceprintSHA512,
		pub.PublicKey,
		pub.Status,
		pub.RegistryStatus,
		pub.Source,
		pub.Realness,
		pub.EnrollmentCenter,
	}, " "))
	return strings.Contains(haystack, q)
}

// Revoke marks an official as revoked. Keeps the embedding so you can audit
// past matches against it, but Match should refuse to verify a revoked
// official.
func (s *enrollmentStore) Revoke(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.by_id[id]
	if !ok {
		return fmt.Errorf("no such official: %s", id)
	}
	now := time.Now().UTC()
	o.Status = "revoked"
	o.RevokedAt = &now
	// Re-sign the revoked record
	canon := *o
	canon.Signature = ""
	canon.Embedding = nil
	env, err := s.signing.Sign(canon)
	if err != nil {
		return fmt.Errorf("sign revocation: %w", err)
	}
	o.Signature = env.Signature
	return s.persist()
}

func (s *enrollmentStore) persist() error {
	tmp := s.path + ".tmp"
	all := make([]*Official, 0, len(s.by_id))
	for _, o := range s.by_id {
		all = append(all, o)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *enrollmentStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	var arr []*Official
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}
	for _, o := range arr {
		s.by_id[o.ID] = o
	}
	return nil
}
