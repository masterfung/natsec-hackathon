// Bounded in-memory ringbuffer of recent signed events. Powers /verify/:id and
// the live SOC feed. We don't persist these; they're a 2,000-event audit
// window. Long-term audit lives in Foundry RouteAlertComments.
package main

import (
	"sync"
	"time"
)

type AuthEventRecord struct {
	ID                string          `json:"id"`
	Channel           string          `json:"channel"` // voice | text | image | document | output
	Verdict           string          `json:"verdict"` // VERIFIED | WARN | BLOCKED
	VerdictReason     string          `json:"verdict_reason,omitempty"`
	SenderClaimed     string          `json:"sender_claimed,omitempty"`
	OfficialID        string          `json:"official_id,omitempty"`
	OfficialName      string          `json:"official_name,omitempty"`
	SpeakerMatch      *float64        `json:"speaker_match,omitempty"`
	DeepfakeRisk      *int            `json:"deepfake_risk,omitempty"`
	InjectionRisk     *int            `json:"injection_risk,omitempty"`
	UltrasonicRisk    *int            `json:"ultrasonic_risk,omitempty"`
	OOBResponse       string          `json:"oob_response,omitempty"`
	OOBPromptID       string          `json:"oob_prompt_id,omitempty"`
	OverallRisk       int             `json:"overall_risk"`
	LatencyMs         int             `json:"latency_ms"`
	Signals           any             `json:"signals,omitempty"`
	Transcript        string          `json:"transcript,omitempty"`
	ProcessedAt       time.Time       `json:"processed_at"`
	IssuerFingerprint string          `json:"issuer_fingerprint"`
	SignedEnvelope    *signedEnvelope `json:"signed_envelope,omitempty"`
}

type eventStore struct {
	mu       sync.RWMutex
	capacity int
	ring     []AuthEventRecord
	byID     map[string]int
	next     int
	count    int
}

func newEventStore(capacity int) *eventStore {
	if capacity < 16 {
		capacity = 16
	}
	return &eventStore{
		capacity: capacity,
		ring:     make([]AuthEventRecord, capacity),
		byID:     make(map[string]int, capacity),
	}
}

// Add stores a record and returns its index.
func (s *eventStore) Add(rec AuthEventRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.next
	if old := s.ring[idx]; old.ID != "" {
		delete(s.byID, old.ID)
	}
	s.ring[idx] = rec
	s.byID[rec.ID] = idx
	s.next = (s.next + 1) % s.capacity
	if s.count < s.capacity {
		s.count++
	}
}

func (s *eventStore) Get(id string) (AuthEventRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	idx, ok := s.byID[id]
	if !ok {
		return AuthEventRecord{}, false
	}
	return s.ring[idx], true
}

// Recent returns the last n events newest-first.
func (s *eventStore) Recent(n int) []AuthEventRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if n <= 0 || n > s.count {
		n = s.count
	}
	out := make([]AuthEventRecord, 0, n)
	// Walk backward from next-1
	for i := 0; i < n; i++ {
		idx := (s.next - 1 - i + s.capacity) % s.capacity
		out = append(out, s.ring[idx])
	}
	return out
}
