// WebAuthn passkey support — second factor that proves the signer holds a
// device with a private key sealed in the platform authenticator (Secure
// Enclave / TPM / hardware token). Unlike the Ed25519 signing key the user
// downloads on enrollment, this private key is non-exfiltratable by design.
//
// Two ceremonies:
//   - Registration: official_id binds a new credential to the registry record.
//   - Assertion:    audio_sha512 is bound to the WebAuthn session; both the
//                   passkey signature AND the audio hash must match.
//
// The library calls the device's private key — we never see it. We persist
// only the credential ID + public key + sign counter on the Official record.
package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

// webauthnService wraps the library, the session store, and the rules used
// when serializing options to the browser.
type webauthnService struct {
	wa       *webauthn.WebAuthn
	sessions sync.Map // session_id -> *webauthnSession
}

type webauthnSession struct {
	OfficialID  string
	Kind        string // "register" | "assert"
	AudioSHA512 string // assert only
	Data        webauthn.SessionData
	ExpiresAt   time.Time
}

func newWebAuthnService(rpID, rpDisplayName string, rpOrigins []string) (*webauthnService, error) {
	if rpID == "" {
		rpID = "localhost"
	}
	if rpDisplayName == "" {
		rpDisplayName = "Mighty Morphing Trust Gate"
	}
	if len(rpOrigins) == 0 {
		rpOrigins = []string{"http://localhost:5173", "http://localhost:7000"}
	}
	wa, err := webauthn.New(&webauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpDisplayName,
		RPOrigins:     rpOrigins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthn init: %w", err)
	}
	return &webauthnService{wa: wa}, nil
}

func (svc *webauthnService) putSession(id string, sess *webauthnSession) {
	sess.ExpiresAt = time.Now().Add(2 * time.Minute)
	svc.sessions.Store(id, sess)
}

func (svc *webauthnService) takeSession(id string) (*webauthnSession, bool) {
	v, ok := svc.sessions.LoadAndDelete(id)
	if !ok {
		return nil, false
	}
	sess, ok := v.(*webauthnSession)
	if !ok {
		return nil, false
	}
	if time.Now().After(sess.ExpiresAt) {
		return nil, false
	}
	return sess, true
}

// webauthnUser adapts an Official into the webauthn.User interface.
type webauthnUser struct {
	off *Official
}

func (u *webauthnUser) WebAuthnID() []byte                         { return []byte(u.off.ID) }
func (u *webauthnUser) WebAuthnName() string                       { return u.off.ID }
func (u *webauthnUser) WebAuthnDisplayName() string                { return u.off.Name }
func (u *webauthnUser) WebAuthnIcon() string                       { return "" } //nolint:staticcheck
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential { return u.off.Passkeys }

// ----------------------------------------------------------------------------
// /webauthn/register/begin
// ----------------------------------------------------------------------------

type webauthnBeginRequest struct {
	OfficialID  string `json:"official_id"`
	AudioSHA512 string `json:"audio_sha512,omitempty"`
}

type webauthnBeginResponse struct {
	SessionID string `json:"session_id"`
	Options   any    `json:"options"`
}

func (s *server) handleWebAuthnRegisterBegin(w http.ResponseWriter, r *http.Request) {
	if s.webauthn == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webauthn not configured"})
		return
	}
	var req webauthnBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decode: " + err.Error()})
		return
	}
	off, ok := s.enrollments.Get(strings.TrimSpace(req.OfficialID))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	user := &webauthnUser{off: off}
	options, sessionData, err := s.webauthn.wa.BeginRegistration(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin registration: " + err.Error()})
		return
	}
	sessID := newEventID()
	s.webauthn.putSession(sessID, &webauthnSession{
		OfficialID: off.ID,
		Kind:       "register",
		Data:       *sessionData,
	})
	writeJSON(w, http.StatusOK, webauthnBeginResponse{SessionID: sessID, Options: options})
}

// ----------------------------------------------------------------------------
// /webauthn/register/finish
// ----------------------------------------------------------------------------

type webauthnFinishRegisterRequest struct {
	SessionID  string          `json:"session_id"`
	Credential json.RawMessage `json:"credential"`
}

func (s *server) handleWebAuthnRegisterFinish(w http.ResponseWriter, r *http.Request) {
	if s.webauthn == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webauthn not configured"})
		return
	}
	var req webauthnFinishRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decode: " + err.Error()})
		return
	}
	sess, ok := s.webauthn.takeSession(strings.TrimSpace(req.SessionID))
	if !ok || sess.Kind != "register" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session not found or expired"})
		return
	}
	off, ok := s.enrollments.Get(sess.OfficialID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(strings.NewReader(string(req.Credential)))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse credential: " + err.Error()})
		return
	}
	user := &webauthnUser{off: off}
	cred, err := s.webauthn.wa.CreateCredential(user, sess.Data, parsed)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "create credential: " + err.Error()})
		return
	}
	if err := s.enrollments.AppendPasskey(off.ID, *cred); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "persist passkey: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":            true,
		"official_id":   off.ID,
		"credential_id": parsed.ID,
		"passkey_count": len(off.Passkeys) + 1,
	})
}

// ----------------------------------------------------------------------------
// /webauthn/assert/begin — bind audio_sha512 to this passkey ceremony
// ----------------------------------------------------------------------------

func (s *server) handleWebAuthnAssertBegin(w http.ResponseWriter, r *http.Request) {
	if s.webauthn == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webauthn not configured"})
		return
	}
	var req webauthnBeginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decode: " + err.Error()})
		return
	}
	audioHash := strings.ToLower(strings.TrimSpace(req.AudioSHA512))
	if audioHash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "audio_sha512 required to bind the assertion"})
		return
	}
	off, ok := s.enrollments.Get(strings.TrimSpace(req.OfficialID))
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	if len(off.Passkeys) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no passkeys enrolled for this official"})
		return
	}
	user := &webauthnUser{off: off}
	options, sessionData, err := s.webauthn.wa.BeginLogin(user)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "begin login: " + err.Error()})
		return
	}
	sessID := newEventID()
	s.webauthn.putSession(sessID, &webauthnSession{
		OfficialID:  off.ID,
		Kind:        "assert",
		AudioSHA512: audioHash,
		Data:        *sessionData,
	})
	writeJSON(w, http.StatusOK, webauthnBeginResponse{SessionID: sessID, Options: options})
}

// ----------------------------------------------------------------------------
// /webauthn/assert/finish — verify the assertion and the audio binding
// ----------------------------------------------------------------------------

type webauthnFinishAssertRequest struct {
	SessionID   string          `json:"session_id"`
	Credential  json.RawMessage `json:"credential"`
	AudioSHA512 string          `json:"audio_sha512"`
}

type webauthnAssertResult struct {
	OK              bool     `json:"ok"`
	OfficialID      string   `json:"official_id"`
	AudioSHA512     string   `json:"audio_sha512"`
	CredentialID    string   `json:"credential_id"`
	UserVerified    bool     `json:"user_verified"`
	UserPresent     bool     `json:"user_present"`
	WarningMessages []string `json:"warnings,omitempty"`
	Error           string   `json:"error,omitempty"`
}

func (s *server) handleWebAuthnAssertFinish(w http.ResponseWriter, r *http.Request) {
	if s.webauthn == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "webauthn not configured"})
		return
	}
	var req webauthnFinishAssertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decode: " + err.Error()})
		return
	}
	sess, ok := s.webauthn.takeSession(strings.TrimSpace(req.SessionID))
	if !ok || sess.Kind != "assert" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "session not found or expired"})
		return
	}
	off, ok := s.enrollments.Get(sess.OfficialID)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	clientAudio := strings.ToLower(strings.TrimSpace(req.AudioSHA512))
	if clientAudio == "" || clientAudio != sess.AudioSHA512 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "audio_sha512 mismatch: the assertion was bound to a different audio file",
		})
		return
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(strings.NewReader(string(req.Credential)))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse credential: " + err.Error()})
		return
	}
	user := &webauthnUser{off: off}
	cred, err := s.webauthn.wa.ValidateLogin(user, sess.Data, parsed)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, webauthnAssertResult{
			OK:          false,
			OfficialID:  off.ID,
			AudioSHA512: clientAudio,
			Error:       "validate login: " + err.Error(),
		})
		return
	}
	// Persist updated counter to detect cloned authenticators on next use.
	if err := s.enrollments.UpdatePasskeyCounter(off.ID, cred); err != nil {
		// Non-fatal for the demo, but worth flagging.
		writeJSON(w, http.StatusOK, webauthnAssertResult{
			OK:              true,
			OfficialID:      off.ID,
			AudioSHA512:     clientAudio,
			CredentialID:    encodeBase64URL(cred.ID),
			UserVerified:    cred.Flags.UserVerified,
			UserPresent:     cred.Flags.UserPresent,
			WarningMessages: []string{"counter persistence warning: " + err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, webauthnAssertResult{
		OK:           true,
		OfficialID:   off.ID,
		AudioSHA512:  clientAudio,
		CredentialID: encodeBase64URL(cred.ID),
		UserVerified: cred.Flags.UserVerified,
		UserPresent:  cred.Flags.UserPresent,
	})
}

func encodeBase64URL(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// ----------------------------------------------------------------------------
// /webauthn/status/{id} — UI uses this to show whether a passkey is enrolled
// ----------------------------------------------------------------------------

type webauthnStatusResponse struct {
	OfficialID    string   `json:"official_id"`
	Enabled       bool     `json:"enabled"`
	PasskeyCount  int      `json:"passkey_count"`
	CredentialIDs []string `json:"credential_ids,omitempty"`
}

func (s *server) handleWebAuthnStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	off, ok := s.enrollments.Get(id)
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such official"})
		return
	}
	creds := make([]string, 0, len(off.Passkeys))
	for _, c := range off.Passkeys {
		creds = append(creds, encodeBase64URL(c.ID))
	}
	writeJSON(w, http.StatusOK, webauthnStatusResponse{
		OfficialID:    off.ID,
		Enabled:       s.webauthn != nil,
		PasskeyCount:  len(off.Passkeys),
		CredentialIDs: creds,
	})
}

// AppendPasskey adds a new credential to the official's record and persists.
func (s *enrollmentStore) AppendPasskey(id string, cred webauthn.Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.by_id[id]
	if !ok {
		return errors.New("no such official")
	}
	o.Passkeys = append(o.Passkeys, cred)
	return s.persist()
}

// UpdatePasskeyCounter replaces the credential whose ID matches with the
// updated one (sign count + cloned-detection fields).
func (s *enrollmentStore) UpdatePasskeyCounter(id string, cred *webauthn.Credential) error {
	if cred == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.by_id[id]
	if !ok {
		return errors.New("no such official")
	}
	for i := range o.Passkeys {
		if string(o.Passkeys[i].ID) == string(cred.ID) {
			o.Passkeys[i] = *cred
			return s.persist()
		}
	}
	return nil
}
