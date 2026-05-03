package main

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type auditEntry struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	Timestamp      string          `json:"ts"`
	ActorID        string          `json:"actor_id,omitempty"`
	Fingerprint    string          `json:"fingerprint,omitempty"`
	PrevHash       string          `json:"prev_hash"`
	Hash           string          `json:"hash"`
	Payload        json.RawMessage `json:"payload"`
	ServerEnvelope *signedEnvelope `json:"server_envelope,omitempty"`
}

type auditStore struct {
	path string
	mu   sync.RWMutex
	sk   *signingKey
	log  []auditEntry
}

func newAuditStore(path string, sk *signingKey) (*auditStore, error) {
	if path == "" {
		path = "data/audit.jsonl"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("mkdir audit data: %w", err)
	}
	s := &auditStore{path: path, sk: sk}
	if err := s.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}

func (s *auditStore) Append(eventType, actorID, fingerprint string, payload any) (auditEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, err := json.Marshal(payload)
	if err != nil {
		return auditEntry{}, fmt.Errorf("marshal audit payload: %w", err)
	}
	prev := ""
	if len(s.log) > 0 {
		prev = s.log[len(s.log)-1].Hash
	}
	entry := auditEntry{
		ID:          newEventID(),
		Type:        eventType,
		Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
		ActorID:     actorID,
		Fingerprint: fingerprint,
		PrevHash:    prev,
		Payload:     body,
	}
	hash, err := auditEntryHash(entry)
	if err != nil {
		return auditEntry{}, err
	}
	entry.Hash = hash
	env, err := s.sk.Sign(struct {
		ID       string          `json:"id"`
		Type     string          `json:"type"`
		TS       string          `json:"ts"`
		PrevHash string          `json:"prev_hash"`
		Hash     string          `json:"hash"`
		Payload  json.RawMessage `json:"payload"`
	}{
		ID:       entry.ID,
		Type:     entry.Type,
		TS:       entry.Timestamp,
		PrevHash: entry.PrevHash,
		Hash:     entry.Hash,
		Payload:  entry.Payload,
	})
	if err != nil {
		return auditEntry{}, fmt.Errorf("sign audit event: %w", err)
	}
	entry.ServerEnvelope = env

	line, err := json.Marshal(entry)
	if err != nil {
		return auditEntry{}, err
	}
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return auditEntry{}, err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return auditEntry{}, err
	}
	s.log = append(s.log, entry)
	return entry, nil
}

func (s *auditStore) List() []auditEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]auditEntry, len(s.log))
	copy(out, s.log)
	return out
}

func (s *auditStore) ChainValid() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	prev := ""
	for _, entry := range s.log {
		if entry.PrevHash != prev {
			return false
		}
		hash, err := auditEntryHash(entry)
		if err != nil || hash != entry.Hash {
			return false
		}
		prev = entry.Hash
	}
	return true
}

func (s *auditStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry auditEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			return fmt.Errorf("decode audit entry: %w", err)
		}
		s.log = append(s.log, entry)
	}
	return nil
}

func auditEntryHash(entry auditEntry) (string, error) {
	canon := struct {
		ID          string          `json:"id"`
		Type        string          `json:"type"`
		Timestamp   string          `json:"ts"`
		ActorID     string          `json:"actor_id,omitempty"`
		Fingerprint string          `json:"fingerprint,omitempty"`
		PrevHash    string          `json:"prev_hash"`
		Payload     json.RawMessage `json:"payload"`
	}{
		ID:          entry.ID,
		Type:        entry.Type,
		Timestamp:   entry.Timestamp,
		ActorID:     entry.ActorID,
		Fingerprint: entry.Fingerprint,
		PrevHash:    entry.PrevHash,
		Payload:     entry.Payload,
	}
	b, err := json.Marshal(canon)
	if err != nil {
		return "", err
	}
	sum := sha512.Sum512(b)
	return hex.EncodeToString(sum[:]), nil
}

func splitLines(data []byte) [][]byte {
	var out [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			out = append(out, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		out = append(out, data[start:])
	}
	return out
}
