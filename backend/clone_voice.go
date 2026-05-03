// Live Cartesia voice-clone-and-return endpoint. Used by the Clone Lab UI to
// take a recorded sample, clone the voice via Cartesia /voices/clone, then
// generate a cloned utterance via /tts/bytes, and stream the resulting WAV
// back to the browser for download. Lets us demo the layered defense by
// producing a real clone of the operator's own voice, signing it with their
// credential, and watching the verifier reject it on deepfake / voiceprint.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"
)

const defaultCloneAttackUtterance = "Authorize wire transfer of fifty thousand dollars to vendor account 4471. This is urgent."

func (s *server) handleVoiceClone(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(strings.TrimSpace(os.Getenv("MM_ENABLE_LIVE_CLONE"))) != "true" {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "live cloning is disabled; set MM_ENABLE_LIVE_CLONE=true for controlled consented demos.",
		})
		return
	}
	apiKey := strings.TrimSpace(os.Getenv("CARTESIA_API_KEY"))
	if apiKey == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "CARTESIA_API_KEY not set on the backend; live cloning disabled.",
		})
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parse multipart: " + err.Error()})
		return
	}
	if strings.TrimSpace(r.FormValue("consent")) != "own_voice" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "live cloning requires explicit consent=own_voice for the uploaded sample.",
		})
		return
	}
	audioBytes, audioFilename, err := readMultipartFile(r, "audio")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if len(audioBytes) < 16_000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("audio too short (%d bytes); record at least ~5 seconds", len(audioBytes)),
		})
		return
	}
	utterance := strings.TrimSpace(r.FormValue("utterance"))
	if utterance == "" {
		utterance = defaultCloneAttackUtterance
	}
	apiBase := strings.TrimRight(getenv("CARTESIA_API_BASE", "https://api.cartesia.ai"), "/")
	apiVersion := getenv("CARTESIA_VERSION", "2026-03-01")
	modelID := getenv("CARTESIA_MODEL_ID", "sonic-3")
	cloneMode := getenv("CARTESIA_CLONE_MODE", "similarity")

	// Step 1 — clone the voice from the uploaded sample.
	voiceName := fmt.Sprintf("mm-live-%d", time.Now().UnixNano())
	cloneBody, err := cartesiaCloneVoice(r.Context(), apiBase, apiKey, apiVersion, voiceName, audioBytes, audioFilename, cloneMode)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "cartesia clone failed: " + err.Error()})
		return
	}
	voiceID, _ := cloneBody["id"].(string)
	if voiceID == "" {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error": "cartesia clone returned no voice id",
			"body":  fmt.Sprintf("%v", cloneBody),
		})
		return
	}

	// Step 2 — synthesise the attack utterance with the cloned voice.
	wavBytes, err := cartesiaSynthesise(r.Context(), apiBase, apiKey, apiVersion, modelID, voiceID, utterance)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error":    "cartesia synthesise failed: " + err.Error(),
			"voice_id": voiceID,
		})
		return
	}
	clonedFilename := fmt.Sprintf("cartesia_clone_%s.wav", voiceID)
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Content-Disposition", `attachment; filename="`+clonedFilename+`"`)
	w.Header().Set("X-Cartesia-Voice-Id", voiceID)
	w.Header().Set("X-Cartesia-Utterance", utterance)
	w.Header().Set("Access-Control-Expose-Headers", "X-Cartesia-Voice-Id, X-Cartesia-Utterance, Content-Disposition")
	_, _ = w.Write(wavBytes)
}

func cartesiaCloneVoice(ctx context.Context, apiBase, apiKey, apiVersion, voiceName string, audio []byte, audioFilename, mode string) (map[string]any, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range map[string]string{
		"name":        voiceName,
		"description": "Mighty Morphing live demo clone (consented sample)",
		"language":    "en",
		"mode":        mode,
		"enhance":     "true",
	} {
		if err := mw.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	clip, err := mw.CreateFormFile("clip", filenameOrDefault(audioFilename, "sample.wav"))
	if err != nil {
		return nil, err
	}
	if _, err := clip.Write(audio); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/voices/clone", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Cartesia-Version", apiVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(respBody), 400))
	}
	var out map[string]any
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, fmt.Errorf("decode clone response: %w (body=%s)", err, truncate(string(respBody), 400))
	}
	return out, nil
}

func cartesiaSynthesise(ctx context.Context, apiBase, apiKey, apiVersion, modelID, voiceID, utterance string) ([]byte, error) {
	payload := map[string]any{
		"model_id":   modelID,
		"transcript": utterance,
		"voice":      map[string]string{"mode": "id", "id": voiceID},
		"output_format": map[string]any{
			"container":   "wav",
			"encoding":    "pcm_s16le",
			"sample_rate": 16000,
		},
		"language": "en",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiBase+"/tts/bytes", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Cartesia-Version", apiVersion)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(body), 400))
	}
	return io.ReadAll(resp.Body)
}

func filenameOrDefault(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fallback
	}
	return name
}
