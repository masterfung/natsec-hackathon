// Mighty Morphing Gateway — Go 1.26 + stdlib net/http
//
// Holds CITADEL_API_KEY, FOUNDRY_TOKEN, and the server's Ed25519 signing key
// server-side. Orchestrates voice-biometric sidecar + Mighty Citadel + Foundry
// actions; broadcasts SSE to the SOC console; emits signed audit envelopes.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ---------------- config ----------------

type config struct {
	BackendPort     string
	LogLevel        string
	CitadelAPIKey   string
	CitadelGateway  string
	CitadelAudio    string
	CitadelAudioWS  string
	CitadelVision   string
	FoundryStackURL string
	FoundryToken    string
	FoundryOntology string
	VoicebioURL     string

	DataDir    string
	SecretsDir string

	SpeakerMatchThreshold float64
	DeepfakeBlockRisk     int
	InjectionBlockRisk    int
	AuthLatencyBudgetMs   int
	OOBTimeoutMs          int
}

func loadConfig() config {
	loadDotenv(".env", "../.env")
	return config{
		// Default 7001 — port 7000 collides with macOS AirPlay Receiver
		// (System Settings → AirDrop & Handoff). Override with MM_BACKEND_PORT.
		BackendPort:     getenv("MM_BACKEND_PORT", "7001"),
		LogLevel:        getenv("MM_LOG_LEVEL", "info"),
		CitadelAPIKey:   os.Getenv("CITADEL_API_KEY"),
		CitadelGateway:  os.Getenv("CITADEL_GATEWAY_URL"),
		CitadelAudio:    os.Getenv("CITADEL_AUDIO_URL"),
		CitadelAudioWS:  os.Getenv("CITADEL_AUDIO_WS_URL"),
		CitadelVision:   os.Getenv("CITADEL_VISION_URL"),
		FoundryStackURL: os.Getenv("FOUNDRY_STACK_URL"),
		FoundryToken:    os.Getenv("FOUNDRY_TOKEN"),
		FoundryOntology: os.Getenv("FOUNDRY_ONTOLOGY_RID"),
		VoicebioURL:     getenv("VOICEBIO_URL", "http://localhost:7100"),

		DataDir:    getenv("MM_DATA_DIR", "../data"),
		SecretsDir: getenv("MM_SECRETS_DIR", "../.secrets"),

		SpeakerMatchThreshold: getenvFloat("MM_SPEAKER_MATCH_THRESHOLD", 0.75),
		DeepfakeBlockRisk:     getenvInt("MM_DEEPFAKE_BLOCK_THRESHOLD", 80),
		InjectionBlockRisk:    getenvInt("MM_INJECTION_BLOCK_THRESHOLD", 70),
		AuthLatencyBudgetMs:   getenvInt("MM_AUTH_LATENCY_BUDGET_MS", 500),
		OOBTimeoutMs:          getenvInt("MM_OOB_TIMEOUT_MS", 5000),
	}
}

// ---------------- main ----------------

func main() {
	cfg := loadConfig()
	setupLogger(cfg.LogLevel)

	sk, err := loadOrCreateSigningKey(cfg.SecretsDir + "/server-ed25519.key")
	if err != nil {
		slog.Error("init signing key", "err", err)
		os.Exit(1)
	}
	voiceprintSalt, err := loadOrCreateSecretBytes(cfg.SecretsDir+"/voiceprint-salt.key", 32)
	if err != nil {
		slog.Error("init voiceprint salt", "err", err)
		os.Exit(1)
	}
	store, err := newEnrollmentStore(cfg.DataDir+"/enrollments.json", sk, voiceprintSalt)
	if err != nil {
		slog.Error("init enrollment store", "err", err)
		os.Exit(1)
	}
	audit, err := newAuditStore(cfg.DataDir+"/audit.jsonl", sk)
	if err != nil {
		slog.Error("init audit store", "err", err)
		os.Exit(1)
	}

	rpID := getenv("MM_WEBAUTHN_RPID", "localhost")
	rpDisplay := getenv("MM_WEBAUTHN_RP_DISPLAY", "Mighty Morphing Trust Gate")
	rpOriginsCSV := getenv("MM_WEBAUTHN_ORIGINS", "http://localhost:5173,http://localhost:7000,http://localhost:7001")
	rpOrigins := []string{}
	for _, o := range strings.Split(rpOriginsCSV, ",") {
		if v := strings.TrimSpace(o); v != "" {
			rpOrigins = append(rpOrigins, v)
		}
	}
	wa, err := newWebAuthnService(rpID, rpDisplay, rpOrigins)
	if err != nil {
		slog.Warn("webauthn disabled", "err", err)
	}

	s := &server{
		cfg:         cfg,
		voice:       newVoiceClient(cfg.VoicebioURL),
		citadel:     newCitadelClient(cfg),
		foundry:     newFoundryClient(cfg),
		sse:         newSSEHub(),
		enrollments: store,
		signing:     sk,
		events:      newEventStore(2000),
		audit:       audit,
		oob:         newOOBManager(time.Duration(cfg.OOBTimeoutMs) * time.Millisecond),
		webauthn:    wa,
	}

	startedAt := time.Now()
	mux := http.NewServeMux()

	// liveness + config
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":             "ok",
			"uptime_s":           int(time.Since(startedAt).Seconds()),
			"citadel_configured": cfg.CitadelAPIKey != "" && cfg.CitadelAudio != "",
			"foundry_configured": s.foundry.Configured(),
			"voicebio_url":       cfg.VoicebioURL,
			"signing_issuer":     sk.FingerprintShort(),
			"enrolled_count":     len(store.List()),
		})
	})
	mux.HandleFunc("GET /api/config-check", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"citadel_gateway":         cfg.CitadelGateway,
			"citadel_audio":           cfg.CitadelAudio,
			"citadel_vision":          cfg.CitadelVision,
			"foundry_stack":           cfg.FoundryStackURL,
			"foundry_ontology":        cfg.FoundryOntology,
			"voicebio_url":            cfg.VoicebioURL,
			"has_citadel_key":         cfg.CitadelAPIKey != "",
			"has_foundry_token":       cfg.FoundryToken != "",
			"speaker_match_threshold": cfg.SpeakerMatchThreshold,
			"deepfake_block_risk":     cfg.DeepfakeBlockRisk,
			"injection_block_risk":    cfg.InjectionBlockRisk,
			"auth_latency_budget_ms":  cfg.AuthLatencyBudgetMs,
			"oob_timeout_ms":          cfg.OOBTimeoutMs,
			"signing_issuer":          sk.FingerprintShort(),
		})
	})

	// voice biometric: enrollment + 2-factor auth + lifecycle
	mux.HandleFunc("POST /api/enroll", s.handleEnroll)
	mux.HandleFunc("POST /api/auth", s.handleAuth)
	mux.HandleFunc("POST /api/verify", s.handleGatewayVerify)
	mux.HandleFunc("POST /demo/gateway/{kind}", s.handleDemoGatewayVerify)
	mux.HandleFunc("GET /api/officials", s.handleListOfficials)
	mux.HandleFunc("GET /api/officials/{id}", s.handleGetOfficial)
	mux.HandleFunc("POST /api/officials/{id}/revoke", s.handleRevokeOfficial)
	mux.HandleFunc("GET /ws/client", s.handleOOBWS)

	// Layer 1 open registry + detached media signing protocol.
	mux.HandleFunc("POST /registry/enroll", s.handleRegistryEnroll)
	mux.HandleFunc("GET /registry/officials", s.handleRegistryOfficials)
	mux.HandleFunc("GET /registry/officials/{id}", s.handleRegistryOfficial)
	mux.HandleFunc("GET /registry/search", s.handleRegistrySearch)
	mux.HandleFunc("GET /registry/lookup", s.handleRegistryLookup)
	mux.HandleFunc("GET /registry/identity/{id}", s.handleRegistryIdentity)
	mux.HandleFunc("POST /registry/officials/{id}/revoke", s.handleRevokeOfficial)
	mux.HandleFunc("POST /sign", s.handleSignMedia)
	mux.HandleFunc("POST /verify", s.handleVerifyMedia)
	mux.HandleFunc("GET /audit", s.handleAudit)
	mux.HandleFunc("GET /demo/fixtures", s.handleDemoFixtures)
	mux.HandleFunc("GET /demo/clone-eval", s.handleDemoCloneEval)
	mux.HandleFunc("GET /demo/fixture-audio", s.handleDemoFixtureAudio)
	mux.HandleFunc("POST /api/voice/clone", s.handleVoiceClone)

	// WebAuthn passkey ceremonies — second factor whose private key never
	// leaves the user's authenticator. Audio-bound on assertion.
	mux.HandleFunc("POST /webauthn/register/begin", s.handleWebAuthnRegisterBegin)
	mux.HandleFunc("POST /webauthn/register/finish", s.handleWebAuthnRegisterFinish)
	mux.HandleFunc("POST /webauthn/assert/begin", s.handleWebAuthnAssertBegin)
	mux.HandleFunc("POST /webauthn/assert/finish", s.handleWebAuthnAssertFinish)
	mux.HandleFunc("GET /webauthn/status/{id}", s.handleWebAuthnStatus)

	// AI agent input/output guardrails
	mux.HandleFunc("POST /api/scan/text", s.handleScanText)
	mux.HandleFunc("POST /api/scan/image", s.handleScanImage)
	mux.HandleFunc("POST /api/scan/document", s.handleScanDocument)
	mux.HandleFunc("POST /api/scan/output", s.handleScanOutput)

	// Live console
	mux.HandleFunc("GET /api/events", s.handleEventsSSE)
	mux.HandleFunc("GET /api/events/recent", s.handleRecentEvents)

	// Offline-verifiable audit trail (powers QR-code provenance UX)
	mux.HandleFunc("GET /verify/pubkey", s.handleVerifyPubkey)
	mux.HandleFunc("GET /verify/{id}", s.handleVerifyEvent)

	srv := &http.Server{
		Addr:              ":" + cfg.BackendPort,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("mighty-morphing gateway listening",
			"addr", srv.Addr,
			"signing_issuer", sk.FingerprintShort(),
			"citadel_configured", cfg.CitadelAPIKey != "",
			"foundry_configured", s.foundry.Configured(),
			"voicebio_url", cfg.VoicebioURL,
			"enrolled_count", len(store.List()),
		)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

// ---------------- helpers (used across files) ----------------

func setupLogger(level string) {
	var l slog.Level
	switch strings.ToLower(level) {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getenvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

// loadDotenv is a tiny .env reader: KEY=VALUE per line, # comments, simple
// quote stripping. Skips keys already present in the environment so the real
// shell env always wins.
func loadDotenv(paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 4096)
		for {
			n, err := f.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				break
			}
		}
		_ = f.Close()
		for _, line := range strings.Split(string(buf), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			eq := strings.IndexByte(line, '=')
			if eq < 0 {
				continue
			}
			k := strings.TrimSpace(line[:eq])
			v := strings.TrimSpace(line[eq+1:])
			if len(v) >= 2 {
				if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
					v = v[1 : len(v)-1]
				}
			}
			if _, ok := os.LookupEnv(k); !ok {
				_ = os.Setenv(k, v)
			}
		}
	}
}
