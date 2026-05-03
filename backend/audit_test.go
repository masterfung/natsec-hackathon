package main

import (
	"testing"
)

func TestAuditStoreHashChainPersistsAndDetectsTamper(t *testing.T) {
	dir := t.TempDir()
	sk, err := loadOrCreateSigningKey(dir + "/server.key")
	if err != nil {
		t.Fatalf("signing key: %v", err)
	}
	store, err := newAuditStore(dir+"/audit.jsonl", sk)
	if err != nil {
		t.Fatalf("audit store: %v", err)
	}
	if _, err := store.Append("one", "actor", "fingerprint", map[string]string{"x": "1"}); err != nil {
		t.Fatalf("append one: %v", err)
	}
	if _, err := store.Append("two", "actor", "fingerprint", map[string]string{"x": "2"}); err != nil {
		t.Fatalf("append two: %v", err)
	}
	if !store.ChainValid() {
		t.Fatalf("expected chain to validate")
	}

	reloaded, err := newAuditStore(dir+"/audit.jsonl", sk)
	if err != nil {
		t.Fatalf("reload audit store: %v", err)
	}
	if !reloaded.ChainValid() {
		t.Fatalf("expected reloaded chain to validate")
	}
	reloaded.log[1].Payload = []byte(`{"x":"tampered"}`)
	if reloaded.ChainValid() {
		t.Fatalf("expected tampered payload to invalidate chain")
	}
}
