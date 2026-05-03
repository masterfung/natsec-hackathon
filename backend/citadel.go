// Mighty Citadel API client (audio scan, vision scan, text scan)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type citadelClient struct {
	gatewayURL string
	audioURL   string
	visionURL  string
	apiKey     string
	http       *http.Client
}

func newCitadelClient(cfg config) *citadelClient {
	return &citadelClient{
		gatewayURL: strings.TrimRight(cfg.CitadelGateway, "/"),
		audioURL:   strings.TrimRight(cfg.CitadelAudio, "/"),
		visionURL:  strings.TrimRight(cfg.CitadelVision, "/"),
		apiKey:     cfg.CitadelAPIKey,
		http:       &http.Client{Timeout: 30 * time.Second},
	}
}

// audioScanResult mirrors Citadel /v1/audio/scan response. Only the fields we
// consume are typed; extra signal fields are kept as raw JSON for forensic
// trail.
type audioScanResult struct {
	Action        string          `json:"action"` // ALLOW | WARN | BLOCK
	Risk          int             `json:"risk"`
	Transcript    string          `json:"transcript,omitempty"`
	DeepfakeRisk  int             `json:"deepfake_risk,omitempty"`
	InjectionRisk int             `json:"injection_risk,omitempty"`
	UltrasonicRsk int             `json:"ultrasonic_risk,omitempty"`
	Signals       any             `json:"signals,omitempty"`
	ProcessingMs  int             `json:"processing_ms,omitempty"`
	Raw           json.RawMessage `json:"-"`
}

// ScanAudio sends an audio buffer to Citadel /v1/audio/scan.
// mode: "fast" | "secure" | "comprehensive"
func (c *citadelClient) ScanAudio(ctx context.Context, audioBytes []byte, filename, mode string) (*audioScanResult, error) {
	if c.audioURL == "" {
		return nil, fmt.Errorf("CITADEL_AUDIO_URL not configured")
	}
	if mode == "" {
		mode = "secure"
	}
	if filename == "" {
		filename = "audio.wav"
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if part, err := mw.CreateFormFile("file", filename); err != nil {
		return nil, err
	} else if _, err := part.Write(audioBytes); err != nil {
		return nil, err
	}
	_ = mw.WriteField("mode", mode)
	if err := mw.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.audioURL+"/v1/audio/scan/file", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("citadel audio: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("citadel audio status=%d body=%s", resp.StatusCode, truncate(string(raw), 400))
	}

	out := &audioScanResult{Raw: raw}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("decode citadel audio: %w", err)
	}
	normalizeAudioScan(out, raw)
	return out, nil
}

func normalizeAudioScan(out *audioScanResult, raw []byte) {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return
	}
	if out.Risk == 0 {
		out.Risk = intFromAny(body["risk_score"])
	}
	if out.ProcessingMs == 0 {
		out.ProcessingMs = intFromAny(body["processing_time_ms"])
	}
	if out.DeepfakeRisk == 0 {
		out.DeepfakeRisk = intFromAny(body["deepfake_risk"])
	}
	if out.DeepfakeRisk == 0 {
		out.DeepfakeRisk = signalRisk(body["signals"], "deepfake_detection")
	}
	if out.DeepfakeRisk == 0 {
		out.DeepfakeRisk = intFromAny(body["deepfake_score"])
	}
	if out.InjectionRisk == 0 {
		out.InjectionRisk = intFromAny(body["injection_risk"])
	}
	if out.InjectionRisk == 0 {
		out.InjectionRisk = max3(
			signalRisk(body["signals"], "asr_prefix_attack"),
			signalRisk(body["signals"], "spoken_prompt_injection"),
			signalRisk(body["signals"], "text_pipeline"),
		)
	}
	if out.UltrasonicRsk == 0 {
		out.UltrasonicRsk = signalRisk(body["signals"], "ultrasonic_injection")
	}
}

func signalRisk(signals any, name string) int {
	arr, ok := signals.([]any)
	if !ok {
		return 0
	}
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if fmt.Sprint(m["name"]) == name {
			return intFromAny(m["risk_score"])
		}
	}
	return 0
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		if x <= 1 {
			return int(x * 100)
		}
		return int(x)
	case json.Number:
		f, _ := x.Float64()
		if f <= 1 {
			return int(f * 100)
		}
		return int(f)
	default:
		return 0
	}
}

// textScanResult — minimal shape for Citadel /v1/scan or gateway scan.
type textScanResult struct {
	Action        string          `json:"action"`
	Risk          int             `json:"risk"`
	Categories    []string        `json:"categories,omitempty"`
	InjectionRisk int             `json:"injection_risk,omitempty"`
	ExfilRisk     int             `json:"exfil_risk,omitempty"`
	Signals       map[string]any  `json:"signals,omitempty"`
	ProcessingMs  int             `json:"processing_ms,omitempty"`
	Raw           json.RawMessage `json:"-"`
}

// ScanText sends arbitrary text to Citadel gateway for prompt-injection /
// exfil / secrets / toxicity scanning.
func (c *citadelClient) ScanText(ctx context.Context, text string, mode string) (*textScanResult, error) {
	if c.gatewayURL == "" {
		return nil, fmt.Errorf("CITADEL_GATEWAY_URL not configured")
	}
	if mode == "" {
		mode = "secure"
	}

	payload, _ := json.Marshal(map[string]any{
		"input":    text,
		"scanMode": mode,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.gatewayURL+"/v1/scan", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("citadel text: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("citadel text status=%d body=%s", resp.StatusCode, truncate(string(raw), 400))
	}
	out := &textScanResult{Raw: raw}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("decode citadel text: %w", err)
	}
	return out, nil
}

// ScanImage sends an image buffer to Citadel vision /v1/scan.
func (c *citadelClient) ScanImage(ctx context.Context, imageBytes []byte, filename, mode string) (*textScanResult, error) {
	if c.visionURL == "" && c.gatewayURL == "" {
		return nil, fmt.Errorf("no Citadel vision endpoint configured")
	}
	target := c.visionURL
	if target == "" {
		target = c.gatewayURL
	}
	if mode == "" {
		mode = "secure"
	}
	if filename == "" {
		filename = "image.png"
	}

	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	if part, err := mw.CreateFormFile("image", filename); err != nil {
		return nil, err
	} else if _, err := part.Write(imageBytes); err != nil {
		return nil, err
	}
	_ = mw.WriteField("scanMode", mode)
	if err := mw.Close(); err != nil {
		return nil, err
	}

	endpoint := target + "/v1/scan"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	c.applyAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("citadel image: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("citadel image status=%d body=%s", resp.StatusCode, truncate(string(raw), 400))
	}
	out := &textScanResult{Raw: raw}
	if err := json.Unmarshal(raw, out); err != nil {
		return nil, fmt.Errorf("decode citadel image: %w", err)
	}
	return out, nil
}

// Health probes the audio service /health for liveness checks.
func (c *citadelClient) AudioHealth(ctx context.Context) (map[string]any, error) {
	if c.audioURL == "" {
		return nil, fmt.Errorf("audio url not configured")
	}
	u, err := url.Parse(c.audioURL + "/health")
	if err != nil {
		return nil, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	c.applyAuth(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out, nil
}

// applyAuth attaches the Citadel API key — header name varies by endpoint;
// most accept `X-API-Key` or `Authorization: Bearer`. We send both.
func (c *citadelClient) applyAuth(req *http.Request) {
	if c.apiKey == "" {
		return
	}
	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	// Audio service in this stack uses X-Internal-Token in some deployments
	req.Header.Set("X-Internal-Token", c.apiKey)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
