// voicebio sidecar client (SpeechBrain ECAPA-TDNN at :7100)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type voiceClient struct {
	baseURL string
	http    *http.Client
}

func newVoiceClient(baseURL string) *voiceClient {
	return &voiceClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

type voiceEmbedResult struct {
	Embedding   []float32 `json:"embedding"`
	Dim         int       `json:"dim"`
	DurationS   float64   `json:"duration_s"`
	Quality     float64   `json:"quality"`
	InferenceMs int       `json:"inference_ms"`
}

// Embed POSTs an audio blob to /embed and returns the speaker embedding.
// audioBytes can be common browser/server audio formats (wav/flac/ogg/aiff/webm/mp4).
func (c *voiceClient) Embed(ctx context.Context, audioBytes []byte, filename string) (*voiceEmbedResult, error) {
	if filename == "" {
		filename = "audio.wav"
	}
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	part, err := mw.CreateFormFile("audio", filename)
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(audioBytes); err != nil {
		return nil, fmt.Errorf("write audio: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embed", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("voicebio /embed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("voicebio /embed status=%d body=%s", resp.StatusCode, string(b))
	}

	var out voiceEmbedResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if out.Dim == 0 || len(out.Embedding) == 0 {
		return nil, errors.New("voicebio returned empty embedding")
	}
	return &out, nil
}

// CosineSimilarity returns cosine similarity in [-1,1].
// Returns 0 for any zero-length input or zero-norm vector.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		x := float64(a[i])
		y := float64(b[i])
		dot += x * y
		na += x * x
		nb += y * y
	}
	denom := math.Sqrt(na) * math.Sqrt(nb)
	if denom < 1e-9 {
		return 0
	}
	return dot / denom
}

// EmbedToCSV serializes a 192-dim embedding to a comma-joined string for
// Foundry storage (Workshop tier doesn't support array<double> properties).
func EmbedToCSV(emb []float32) string {
	parts := make([]string, len(emb))
	for i, v := range emb {
		parts[i] = fmt.Sprintf("%.6f", v)
	}
	return strings.Join(parts, ",")
}

// ParseEmbedCSV is the inverse of EmbedToCSV. Returns nil on parse error.
func ParseEmbedCSV(s string) []float32 {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]float32, len(parts))
	for i, p := range parts {
		var f float32
		if _, err := fmt.Sscanf(strings.TrimSpace(p), "%f", &f); err != nil {
			return nil
		}
		out[i] = f
	}
	return out
}
