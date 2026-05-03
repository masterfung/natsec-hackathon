package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestOOBPromptApproveOverWebSocket(t *testing.T) {
	manager := newOOBManager(2 * time.Second)
	server := httptest.NewServer(httpHandler(manager.ServeWS))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "?official_id=user-self"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	var ready map[string]any
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("read ready: %v", err)
	}

	resultCh := make(chan oobResult, 1)
	go func() {
		resultCh <- manager.Prompt(context.Background(), "user-self", "approve this request", nil)
	}()

	var prompt oobPrompt
	if err := conn.ReadJSON(&prompt); err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	if prompt.PromptID == "" || prompt.Snippet == "" {
		t.Fatalf("malformed prompt: %+v", prompt)
	}
	if err := conn.WriteJSON(oobClientMessage{Type: "oob.response", PromptID: prompt.PromptID, Verdict: "APPROVE"}); err != nil {
		t.Fatalf("write response: %v", err)
	}

	select {
	case res := <-resultCh:
		if res.Verdict != "APPROVE" {
			t.Fatalf("expected approve, got %+v", res)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for OOB result")
	}
}

func TestOOBPromptRequiresIdentityChallenge(t *testing.T) {
	manager := newOOBManager(2 * time.Second)
	identity, err := newIdentityEnvelope("user-self", "http://localhost:7000/registry")
	if err != nil {
		t.Fatalf("identity: %v", err)
	}
	server := httptest.NewServer(httpHandler(func(w http.ResponseWriter, r *http.Request) {
		manager.ServeWSAuthenticated(w, r, "user-self", identity.PublicKey)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	var challenge map[string]string
	if err := conn.ReadJSON(&challenge); err != nil {
		t.Fatalf("read challenge: %v", err)
	}
	priv, err := decodeEd25519PrivateKey(identity.PrivateKey)
	if err != nil {
		t.Fatalf("private key: %v", err)
	}
	signature := ed25519.Sign(priv, []byte(challenge["challenge"]))
	if err := conn.WriteJSON(oobChallengeResponse{
		Type:      "oob.challenge_response",
		Challenge: challenge["challenge"],
		Signature: base64.StdEncoding.EncodeToString(signature),
	}); err != nil {
		t.Fatalf("write challenge response: %v", err)
	}
	var ready map[string]any
	if err := conn.ReadJSON(&ready); err != nil {
		t.Fatalf("read ready: %v", err)
	}
	if ready["type"] != "oob.ready" {
		t.Fatalf("expected ready, got %+v", ready)
	}
}

func TestOOBPromptNoClientFailsClosed(t *testing.T) {
	manager := newOOBManager(10 * time.Millisecond)
	res := manager.Prompt(context.Background(), "missing", "", nil)
	if res.Verdict != "TIMEOUT" || res.Reason == "" {
		t.Fatalf("expected fail-closed timeout, got %+v", res)
	}
}

func httpHandler(fn func(http.ResponseWriter, *http.Request)) http.Handler {
	return http.HandlerFunc(fn)
}
