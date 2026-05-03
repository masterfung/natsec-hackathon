// Palantir Foundry Platform API client (read object types/objects, apply actions)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type foundryClient struct {
	baseURL  string
	token    string
	ontology string // ontology RID
	http     *http.Client
}

func newFoundryClient(cfg config) *foundryClient {
	return &foundryClient{
		baseURL:  strings.TrimRight(cfg.FoundryStackURL, "/"),
		token:    cfg.FoundryToken,
		ontology: cfg.FoundryOntology,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (f *foundryClient) Configured() bool {
	return f.baseURL != "" && f.token != "" && f.ontology != ""
}

// applyActionResult is the response shape for /actions/{name}/apply.
// Foundry returns different shapes for different action operations; we keep it
// loose here.
type applyActionResult struct {
	Status string          `json:"status,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Raw    json.RawMessage `json:"-"`
}

// ApplyAction calls /api/v2/ontologies/{ontology}/actions/{actionApiName}/apply
// with the given parameters. Foundry expects parameters wrapped in a
// {"parameters":{...},"options":{}} body.
func (f *foundryClient) ApplyAction(ctx context.Context, actionAPIName string, parameters map[string]any) (*applyActionResult, error) {
	if !f.Configured() {
		return nil, fmt.Errorf("foundry not configured")
	}
	if actionAPIName == "" {
		return nil, fmt.Errorf("actionAPIName required")
	}

	body := map[string]any{
		"parameters": parameters,
		"options": map[string]any{
			"returnEdits": "ALL",
			"mode":        "VALIDATE_AND_EXECUTE",
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/api/v2/ontologies/%s/actions/%s/apply",
		f.baseURL, f.ontology, actionAPIName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Accept", "application/json")

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("foundry apply-action: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("foundry apply-action %s status=%d body=%s",
			actionAPIName, resp.StatusCode, truncate(string(raw), 400))
	}
	out := &applyActionResult{Raw: raw}
	_ = json.Unmarshal(raw, out)
	return out, nil
}

// ListObjects pages through /api/v2/ontologies/{ontology}/objects/{type}.
// Returns the raw `data` array plus a nextPageToken.
func (f *foundryClient) ListObjects(ctx context.Context, objectType string, pageSize int, pageToken string) ([]map[string]any, string, error) {
	if !f.Configured() {
		return nil, "", fmt.Errorf("foundry not configured")
	}
	if pageSize <= 0 {
		pageSize = 50
	}

	endpoint := fmt.Sprintf("%s/api/v2/ontologies/%s/objects/%s?pageSize=%d",
		f.baseURL, f.ontology, objectType, pageSize)
	if pageToken != "" {
		endpoint += "&pageToken=" + pageToken
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Accept", "application/json")

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("foundry list: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, "", fmt.Errorf("foundry list %s status=%d body=%s",
			objectType, resp.StatusCode, truncate(string(raw), 400))
	}

	var page struct {
		Data          []map[string]any `json:"data"`
		NextPageToken string           `json:"nextPageToken"`
	}
	if err := json.Unmarshal(raw, &page); err != nil {
		return nil, "", fmt.Errorf("decode foundry list: %w", err)
	}
	return page.Data, page.NextPageToken, nil
}

// GetObject fetches a single object by primary key.
func (f *foundryClient) GetObject(ctx context.Context, objectType, primaryKey string) (map[string]any, error) {
	if !f.Configured() {
		return nil, fmt.Errorf("foundry not configured")
	}
	endpoint := fmt.Sprintf("%s/api/v2/ontologies/%s/objects/%s/%s",
		f.baseURL, f.ontology, objectType, primaryKey)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	req.Header.Set("Authorization", "Bearer "+f.token)
	req.Header.Set("Accept", "application/json")

	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("foundry get: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == 404 {
		return nil, nil
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("foundry get %s/%s status=%d body=%s",
			objectType, primaryKey, resp.StatusCode, truncate(string(raw), 400))
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("decode foundry get: %w", err)
	}
	return obj, nil
}

// VerifyOntologySchema checks whether the four MM_ object types and two
// MM_ action types exist in the configured ontology. Returns map of name->exists.
func (f *foundryClient) VerifyOntologySchema(ctx context.Context) (map[string]bool, error) {
	if !f.Configured() {
		return nil, fmt.Errorf("foundry not configured")
	}

	out := map[string]bool{
		"MMOfficial":         false,
		"MMVoiceBiometric":   false,
		"MMCommunication":    false,
		"MMAuthEvent":        false,
		"mm-enroll-official": false,
		"mm-log-auth-event":  false,
	}

	// Object types
	{
		endpoint := fmt.Sprintf("%s/api/v2/ontologies/%s/objectTypes?pageSize=200",
			f.baseURL, f.ontology)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+f.token)
		resp, err := f.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var page struct {
			Data []struct {
				APIName string `json:"apiName"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, err
		}
		for _, t := range page.Data {
			if _, ok := out[t.APIName]; ok {
				out[t.APIName] = true
			}
		}
	}

	// Action types
	{
		endpoint := fmt.Sprintf("%s/api/v2/ontologies/%s/actionTypes?pageSize=200",
			f.baseURL, f.ontology)
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		req.Header.Set("Authorization", "Bearer "+f.token)
		resp, err := f.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		var page struct {
			Data []struct {
				APIName string `json:"apiName"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
			return nil, err
		}
		for _, a := range page.Data {
			if _, ok := out[a.APIName]; ok {
				out[a.APIName] = true
			}
		}
	}

	return out, nil
}
