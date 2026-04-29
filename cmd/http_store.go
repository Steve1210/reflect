package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"reflections/internal"
)

type httpStore struct {
	baseURL string
	client  *http.Client
}

func newHTTPStore(baseURL string) *httpStore {
	return &httpStore{baseURL: baseURL, client: &http.Client{}}
}

func (h *httpStore) InsertReflection(ctx context.Context, r internal.Reflection) (int64, error) {
	body, err := json.Marshal(map[string]any{
		"title": r.Title,
		"tags":  r.Tags,
		"body":  r.Body,
	})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+"/reflections", bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("API returned %d", resp.StatusCode)
	}
	var result internal.Reflection
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Id, nil
}

func (h *httpStore) FetchAllMetadataWithFilters(ctx context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
	params := url.Values{}
	if f.CreatedAfter != 0 {
		params.Set("created_after", unixToDate(f.CreatedAfter))
	}
	if f.CreatedBefore != 0 {
		params.Set("created_before", unixToDate(f.CreatedBefore))
	}
	if f.UpdatedAfter != 0 {
		params.Set("updated_after", unixToDate(f.UpdatedAfter))
	}
	if f.UpdatedBefore != 0 {
		params.Set("updated_before", unixToDate(f.UpdatedBefore))
	}
	u := h.baseURL + "/reflections"
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var results []internal.ReflectionHeader
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, err
	}
	return results, nil
}

func (h *httpStore) FetchReflectionByID(_ context.Context, _ int64) (internal.Reflection, error) {
	return internal.Reflection{}, fmt.Errorf("not supported in API mode")
}

func (h *httpStore) UpdateReflection(_ context.Context, _ int64, _ internal.Reflection) error {
	return fmt.Errorf("not supported in API mode")
}

func (h *httpStore) DeleteReflection(_ context.Context, _ int64) error {
	return fmt.Errorf("not supported in API mode")
}

func unixToDate(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("2006-01-02")
}
