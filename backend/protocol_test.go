package main

import (
	"strings"
	"testing"
)

func TestIdentityEnvelopeAndMediaSignatureRoundTrip(t *testing.T) {
	identity, err := newIdentityEnvelope("user-self", "http://localhost:7000/registry")
	if err != nil {
		t.Fatalf("new identity envelope: %v", err)
	}
	audio := []byte("not real audio but signed bytes")
	sig, err := signMediaEnvelope(identity.ID, "voiceprint", "test", audio, identity.PrivateKey)
	if err != nil {
		t.Fatalf("sign media: %v", err)
	}
	hashMatch, sigValid, err := verifyMediaEnvelope(sig, audio, identity.PublicKey)
	if err != nil {
		t.Fatalf("verify media: %v", err)
	}
	if !hashMatch || !sigValid {
		t.Fatalf("expected hash and signature to pass; hash=%v sig=%v", hashMatch, sigValid)
	}

	hashMatch, sigValid, err = verifyMediaEnvelope(sig, []byte("tampered"), identity.PublicKey)
	if err != nil {
		t.Fatalf("verify tampered media: %v", err)
	}
	if hashMatch || !sigValid {
		t.Fatalf("expected tampered bytes to fail hash only; hash=%v sig=%v", hashMatch, sigValid)
	}

	sig.Signature = strings.Repeat("A", len(sig.Signature))
	_, sigValid, err = verifyMediaEnvelope(sig, audio, identity.PublicKey)
	if err == nil && sigValid {
		t.Fatalf("expected tampered signature to fail")
	}
}

func TestHashEmbeddingSHA512UsesSalt(t *testing.T) {
	emb := []float32{0.1, 0.2, 0.3}
	a := HashEmbeddingSHA512(emb, []byte("salt-a"))
	b := HashEmbeddingSHA512(emb, []byte("salt-b"))
	if len(a) != 128 {
		t.Fatalf("expected SHA-512 hex length, got %d", len(a))
	}
	if a == b {
		t.Fatalf("expected salt to change hash")
	}
}
