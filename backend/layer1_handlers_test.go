package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestLayer1EnrollSignVerifyFlow(t *testing.T) {
	voicebio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embed" {
			http.NotFound(w, r)
			return
		}
		embedding := make([]float32, 192)
		for i := range embedding {
			embedding[i] = float32(i+1) / 192.0
		}
		writeJSON(w, http.StatusOK, voiceEmbedResult{
			Embedding:   embedding,
			Dim:         len(embedding),
			DurationS:   2.5,
			Quality:     0.9,
			InferenceMs: 1,
		})
	}))
	defer voicebio.Close()

	httpServer, cleanup := newLayer1TestHTTPServer(t, voicebio.URL)
	defer cleanup()

	enrollBody := multipartBody(t, map[string]string{
		"id":   "user-self",
		"name": "User Self",
		"role": "demo",
	}, map[string][]byte{
		"audio": []byte("real audio bytes"),
	}, map[string]string{"audio": "real_01.wav"})
	enrollResp, status := postMultipart(t, httpServer.URL+"/registry/enroll", enrollBody)
	if status != http.StatusOK {
		t.Fatalf("enroll status=%d body=%s", status, enrollResp)
	}
	var enrolled enrollResponse
	if err := json.Unmarshal(enrollResp, &enrolled); err != nil {
		t.Fatalf("decode enroll: %v", err)
	}
	if enrolled.IdentityEnvelope == nil || enrolled.IdentityEnvelope.PublicKey == "" {
		t.Fatalf("expected public identity envelope")
	}
	if enrolled.EnrollmentSecret == nil || enrolled.EnrollmentSecret.PrivateKey == "" {
		t.Fatalf("expected one-time enrollment_secret with private key")
	}
	if enrolled.EnrollmentSecret.Warning == "" {
		t.Fatalf("expected enrollment_secret to carry a save-this-now warning")
	}
	// Critical: the public envelope must NOT carry the private key.
	envJSON, _ := json.Marshal(enrolled.IdentityEnvelope)
	if bytes.Contains(envJSON, []byte(enrolled.EnrollmentSecret.PrivateKey)) {
		t.Fatalf("identity_envelope leaked private key in enroll response")
	}
	if enrolled.Official.VoiceprintSHA512 == "" || enrolled.Official.PublicKey == "" {
		t.Fatalf("expected public registry fields in enrollment response")
	}

	regResp, status := get(t, httpServer.URL+"/registry/officials/user-self")
	if status != http.StatusOK {
		t.Fatalf("registry status=%d body=%s", status, regResp)
	}
	if bytes.Contains(regResp, []byte(enrolled.EnrollmentSecret.PrivateKey)) {
		t.Fatalf("registry leaked private key")
	}
	if !bytes.Contains(regResp, []byte(`"source":"local-enrollment"`)) || !bytes.Contains(regResp, []byte(`"realness":"enrolled-real-speaker"`)) {
		t.Fatalf("registry response missing UI metadata labels: %s", regResp)
	}

	searchResp, status := get(t, httpServer.URL+"/registry/search?q=User&status=active&limit=5")
	if status != http.StatusOK {
		t.Fatalf("search status=%d body=%s", status, searchResp)
	}
	if !bytes.Contains(searchResp, []byte(`"id":"user-self"`)) {
		t.Fatalf("expected search to return enrolled profile, got %s", searchResp)
	}

	lookupResp, status := get(t, httpServer.URL+"/registry/lookup?fingerprint="+enrolled.Official.VoiceprintSHA512)
	if status != http.StatusOK {
		t.Fatalf("lookup status=%d body=%s", status, lookupResp)
	}
	if !bytes.Contains(lookupResp, []byte(`"profile"`)) || bytes.Contains(lookupResp, []byte(enrolled.EnrollmentSecret.PrivateKey)) {
		t.Fatalf("lookup response missing profile or leaked private key: %s", lookupResp)
	}

	identityResp, status := get(t, httpServer.URL+"/registry/identity/user-self")
	if status != http.StatusOK {
		t.Fatalf("identity status=%d body=%s", status, identityResp)
	}
	if !bytes.Contains(identityResp, []byte(`"private_key_available":false`)) || bytes.Contains(identityResp, []byte(enrolled.EnrollmentSecret.PrivateKey)) {
		t.Fatalf("public identity response leaked private key or omitted private flag: %s", identityResp)
	}

	duplicateResp, status := postMultipart(t, httpServer.URL+"/registry/enroll", enrollBody)
	if status != http.StatusConflict {
		t.Fatalf("duplicate enroll status=%d body=%s", status, duplicateResp)
	}
	if !bytes.Contains(duplicateResp, []byte(`"code":"official_id_already_active"`)) {
		t.Fatalf("duplicate response missing clear error code: %s", duplicateResp)
	}

	credentialBundle, _ := json.Marshal(map[string]any{
		"identity_envelope": enrolled.IdentityEnvelope,
		"enrollment_secret": enrolled.EnrollmentSecret,
	})
	signBody := multipartBody(t, map[string]string{
		"identity_envelope": string(credentialBundle),
		"purpose":           "test",
	}, map[string][]byte{
		"audio": []byte("real audio bytes"),
	}, map[string]string{"audio": "real_02.wav"})
	signResp, status := postMultipart(t, httpServer.URL+"/sign", signBody)
	if status != http.StatusOK {
		t.Fatalf("sign status=%d body=%s", status, signResp)
	}
	var signed signMediaResponse
	if err := json.Unmarshal(signResp, &signed); err != nil {
		t.Fatalf("decode sign: %v", err)
	}
	sigJSON, _ := json.Marshal(signed.Signature)

	verifyBody := multipartBody(t, nil, map[string][]byte{
		"audio": []byte("real audio bytes"),
		"sig":   sigJSON,
	}, map[string]string{"audio": "real_02.wav", "sig": "real_02.sig.json"})
	verifyResp, status := postMultipart(t, httpServer.URL+"/verify", verifyBody)
	if status != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", status, verifyResp)
	}
	var verified verifyMediaResponse
	if err := json.Unmarshal(verifyResp, &verified); err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if !verified.OK || !verified.SignatureValid || !verified.AudioHashMatch {
		t.Fatalf("expected valid verification, got %+v", verified)
	}
	if verified.VoiceMatch == nil || !*verified.VoiceMatch {
		t.Fatalf("expected positive speaker match, got %+v", verified)
	}

	tamperedBody := multipartBody(t, nil, map[string][]byte{
		"audio": []byte("tampered audio bytes"),
		"sig":   sigJSON,
	}, map[string]string{"audio": "tampered.wav", "sig": "real_02.sig.json"})
	tamperedResp, status := postMultipart(t, httpServer.URL+"/verify", tamperedBody)
	if status != http.StatusOK {
		t.Fatalf("tampered verify status=%d body=%s", status, tamperedResp)
	}
	var tampered verifyMediaResponse
	if err := json.Unmarshal(tamperedResp, &tampered); err != nil {
		t.Fatalf("decode tampered verify: %v", err)
	}
	if tampered.OK || tampered.AudioHashMatch {
		t.Fatalf("expected tampered audio to fail, got %+v", tampered)
	}

	auditResp, status := get(t, httpServer.URL+"/audit")
	if status != http.StatusOK {
		t.Fatalf("audit status=%d body=%s", status, auditResp)
	}
	if !bytes.Contains(auditResp, []byte(`"chain_valid":true`)) {
		t.Fatalf("expected valid audit chain, got %s", auditResp)
	}

	fixturesResp, status := get(t, httpServer.URL+"/demo/fixtures")
	if status != http.StatusOK {
		t.Fatalf("fixtures status=%d body=%s", status, fixturesResp)
	}
	if !bytes.Contains(fixturesResp, []byte(`"policy"`)) || !bytes.Contains(fixturesResp, []byte(`"kind":"clone"`)) {
		t.Fatalf("expected safe fixture metadata, got %s", fixturesResp)
	}
}

func newLayer1TestHTTPServer(t *testing.T, voicebioURL string) (*httptest.Server, func()) {
	t.Helper()
	dir := t.TempDir()
	sk, err := loadOrCreateSigningKey(filepath.Join(dir, "secrets", "server.key"))
	if err != nil {
		t.Fatalf("signing key: %v", err)
	}
	store, err := newEnrollmentStore(filepath.Join(dir, "data", "enrollments.json"), sk, []byte("salt"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	audit, err := newAuditStore(filepath.Join(dir, "data", "audit.jsonl"), sk)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	s := &server{
		cfg: config{
			SpeakerMatchThreshold: 0.75,
			DeepfakeBlockRisk:     80,
			InjectionBlockRisk:    70,
			VoicebioURL:           voicebioURL,
		},
		voice:       newVoiceClient(voicebioURL),
		citadel:     newCitadelClient(config{}),
		foundry:     newFoundryClient(config{}),
		sse:         newSSEHub(),
		enrollments: store,
		signing:     sk,
		events:      newEventStore(16),
		audit:       audit,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /registry/enroll", s.handleRegistryEnroll)
	mux.HandleFunc("GET /registry/search", s.handleRegistrySearch)
	mux.HandleFunc("GET /registry/lookup", s.handleRegistryLookup)
	mux.HandleFunc("GET /registry/officials/{id}", s.handleRegistryOfficial)
	mux.HandleFunc("GET /registry/identity/{id}", s.handleRegistryIdentity)
	mux.HandleFunc("POST /sign", s.handleSignMedia)
	mux.HandleFunc("POST /verify", s.handleVerifyMedia)
	mux.HandleFunc("GET /audit", s.handleAudit)
	mux.HandleFunc("GET /demo/fixtures", s.handleDemoFixtures)
	srv := httptest.NewServer(withCORS(mux))
	return srv, srv.Close
}

type multipartRequestBody struct {
	body        []byte
	contentType string
}

func multipartBody(t *testing.T, fields map[string]string, files map[string][]byte, filenames map[string]string) multipartRequestBody {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	for key, data := range files {
		filename := filenames[key]
		if filename == "" {
			filename = key + ".bin"
		}
		part, err := writer.CreateFormFile(key, filename)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart: %v", err)
	}
	return multipartRequestBody{body: buf.Bytes(), contentType: writer.FormDataContentType()}
}

func postMultipart(t *testing.T, url string, body multipartRequestBody) ([]byte, int) {
	t.Helper()
	resp, err := http.Post(url, body.contentType, bytes.NewReader(body.body))
	if err != nil {
		t.Fatalf("post multipart: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return data, resp.StatusCode
}

func get(t *testing.T, url string) ([]byte, int) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return data, resp.StatusCode
}
