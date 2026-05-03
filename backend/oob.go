package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type oobManager struct {
	mu      sync.RWMutex
	clients map[string]map[*oobClient]struct{}
	pending map[string]chan oobResult
	timeout time.Duration
}

type oobClient struct {
	officialID string
	conn       *websocket.Conn
	send       chan any
	manager    *oobManager
}

type oobPrompt struct {
	Type       string `json:"type"`
	PromptID   string `json:"prompt_id"`
	OfficialID string `json:"official_id"`
	Nonce      string `json:"nonce"`
	Snippet    string `json:"snippet,omitempty"`
	Deadline   string `json:"deadline"`
	Signals    any    `json:"signals,omitempty"`
}

type oobResult struct {
	PromptID string `json:"prompt_id"`
	Verdict  string `json:"verdict"`
	Reason   string `json:"reason,omitempty"`
}

type oobClientMessage struct {
	Type     string `json:"type"`
	PromptID string `json:"prompt_id"`
	Verdict  string `json:"verdict"`
}

type oobChallengeResponse struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Signature string `json:"signature"`
}

func newOOBManager(timeout time.Duration) *oobManager {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &oobManager{
		clients: make(map[string]map[*oobClient]struct{}),
		pending: make(map[string]chan oobResult),
		timeout: timeout,
	}
}

func (m *oobManager) ServeWS(w http.ResponseWriter, r *http.Request) {
	officialID := strings.TrimSpace(r.URL.Query().Get("official_id"))
	if officialID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "official_id query parameter required"})
		return
	}
	m.ServeWSAuthenticated(w, r, officialID, "")
}

func (m *oobManager) ServeWSAuthenticated(w http.ResponseWriter, r *http.Request, officialID, publicKeyHex string) {
	officialID = strings.TrimSpace(officialID)
	if officialID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "official_id required"})
		return
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(*http.Request) bool { return true },
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Warn("oob websocket upgrade failed", "err", err)
		return
	}
	if publicKeyHex != "" {
		if err := authenticateOOBClient(conn, officialID, publicKeyHex); err != nil {
			_ = conn.WriteJSON(map[string]any{"type": "oob.error", "error": err.Error()})
			_ = conn.Close()
			return
		}
	}
	client := &oobClient{
		officialID: officialID,
		conn:       conn,
		send:       make(chan any, 8),
		manager:    m,
	}
	m.addClient(client)
	defer m.removeClient(client)

	go client.writeLoop()
	client.send <- map[string]any{"type": "oob.ready", "official_id": officialID}
	client.readLoop()
}

func authenticateOOBClient(conn *websocket.Conn, officialID, publicKeyHex string) error {
	publicKey, err := decodeEd25519PublicKey(publicKeyHex)
	if err != nil {
		return err
	}
	challenge := newEventID()
	if err := conn.WriteJSON(map[string]any{
		"type":        "oob.challenge",
		"official_id": officialID,
		"challenge":   challenge,
	}); err != nil {
		return err
	}
	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer conn.SetReadDeadline(time.Time{})
	var resp oobChallengeResponse
	if err := conn.ReadJSON(&resp); err != nil {
		return err
	}
	if resp.Type != "oob.challenge_response" || resp.Challenge != challenge {
		return errors.New("invalid OOB challenge response")
	}
	sig, err := base64.StdEncoding.DecodeString(resp.Signature)
	if err != nil {
		return err
	}
	if !ed25519.Verify(publicKey, []byte(challenge), sig) {
		return errors.New("OOB challenge signature verification failed")
	}
	return nil
}

func (m *oobManager) Prompt(ctx context.Context, officialID, snippet string, signals any) oobResult {
	officialID = strings.TrimSpace(officialID)
	if officialID == "" {
		return oobResult{Verdict: "TIMEOUT", Reason: "missing official id"}
	}
	promptID := newEventID()
	deadline := time.Now().Add(m.timeout)
	ch := make(chan oobResult, 1)
	prompt := oobPrompt{
		Type:       "oob.prompt",
		PromptID:   promptID,
		OfficialID: officialID,
		Nonce:      newEventID(),
		Snippet:    truncate(snippet, 240),
		Deadline:   deadline.UTC().Format(time.RFC3339Nano),
		Signals:    signals,
	}

	m.mu.Lock()
	clients := m.clients[officialID]
	if len(clients) == 0 {
		m.mu.Unlock()
		return oobResult{PromptID: promptID, Verdict: "TIMEOUT", Reason: "no phone client connected"}
	}
	m.pending[promptID] = ch
	for client := range clients {
		select {
		case client.send <- prompt:
		default:
			slog.Warn("oob client send buffer full", "official_id", officialID)
		}
	}
	m.mu.Unlock()

	timer := time.NewTimer(m.timeout)
	defer timer.Stop()
	defer m.clearPending(promptID)

	select {
	case res := <-ch:
		return res
	case <-timer.C:
		return oobResult{PromptID: promptID, Verdict: "TIMEOUT", Reason: "OOB-timeout"}
	case <-ctx.Done():
		return oobResult{PromptID: promptID, Verdict: "TIMEOUT", Reason: ctx.Err().Error()}
	}
}

func (m *oobManager) resolve(res oobResult) error {
	if res.PromptID == "" {
		return errors.New("prompt_id required")
	}
	verdict := strings.ToUpper(strings.TrimSpace(res.Verdict))
	if verdict != "APPROVE" && verdict != "DENY" {
		verdict = "DENY"
	}
	res.Verdict = verdict
	m.mu.RLock()
	ch := m.pending[res.PromptID]
	m.mu.RUnlock()
	if ch == nil {
		return errors.New("prompt not pending")
	}
	select {
	case ch <- res:
	default:
	}
	return nil
}

func (m *oobManager) addClient(c *oobClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.clients[c.officialID] == nil {
		m.clients[c.officialID] = make(map[*oobClient]struct{})
	}
	m.clients[c.officialID][c] = struct{}{}
}

func (m *oobManager) removeClient(c *oobClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if clients := m.clients[c.officialID]; clients != nil {
		delete(clients, c)
		if len(clients) == 0 {
			delete(m.clients, c.officialID)
		}
	}
	close(c.send)
	_ = c.conn.Close()
}

func (m *oobManager) clearPending(promptID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.pending, promptID)
}

func (c *oobClient) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			slog.Warn("oob read loop panic", "panic", r)
		}
	}()
	c.conn.SetReadLimit(4096)
	for {
		var msg oobClientMessage
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg.Type == "" {
			msg.Type = "oob.response"
		}
		if msg.Type != "oob.response" {
			continue
		}
		if err := c.manager.resolve(oobResult{PromptID: msg.PromptID, Verdict: msg.Verdict}); err != nil {
			_ = c.conn.WriteJSON(map[string]any{"type": "oob.error", "error": err.Error()})
		}
	}
}

func (c *oobClient) writeLoop() {
	for msg := range c.send {
		if err := c.conn.WriteJSON(msg); err != nil {
			return
		}
	}
}
