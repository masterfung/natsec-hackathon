package main

import "testing"

func TestEnrollmentStoreAddsPublicKeyAndRejectsActiveDuplicate(t *testing.T) {
	dir := t.TempDir()
	sk, err := loadOrCreateSigningKey(dir + "/server.key")
	if err != nil {
		t.Fatalf("signing key: %v", err)
	}
	store, err := newEnrollmentStore(dir+"/enrollments.json", sk, []byte("salt"))
	if err != nil {
		t.Fatalf("enrollment store: %v", err)
	}
	identity, err := newIdentityEnvelope("user-self", "http://localhost:7000/registry")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	off := &Official{
		ID:        identity.ID,
		Name:      "User Self",
		PublicKey: identity.PublicKey,
		Embedding: []float32{0.1, 0.2, 0.3},
	}
	if err := store.Add(off); err != nil {
		t.Fatalf("add enrollment: %v", err)
	}
	got, ok := store.Get(identity.ID)
	if !ok {
		t.Fatalf("missing enrollment")
	}
	if got.PublicKey != identity.PublicKey {
		t.Fatalf("public key not persisted")
	}
	if got.VoiceprintSHA512 == "" || len(got.VoiceprintSHA512) != 128 {
		t.Fatalf("missing SHA-512 voiceprint")
	}
	if err := store.Add(off); err == nil {
		t.Fatalf("expected active duplicate rejection")
	}
	if err := store.Revoke(identity.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	off2 := *off
	off2.Status = "active"
	if err := store.Add(&off2); err != nil {
		t.Fatalf("re-enroll after revoke should pass: %v", err)
	}
}
