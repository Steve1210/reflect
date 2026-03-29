package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"reflections/internal"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// --- mockStore ---

type mockStore struct {
	fetchAllFn  func(ctx context.Context, filters internal.FilterOptions) ([]internal.ReflectionHeader, error)
	fetchByIDFn func(ctx context.Context, id int64) (internal.Reflection, error)
	insertFn    func(ctx context.Context, r internal.Reflection) (int64, error)
	updateFn    func(ctx context.Context, id int64, r internal.Reflection) error
	deleteFn    func(ctx context.Context, id int64) error
}

func (m *mockStore) FetchAllMetadataWithFilters(ctx context.Context, filters internal.FilterOptions) ([]internal.ReflectionHeader, error) {
	return m.fetchAllFn(ctx, filters)
}
func (m *mockStore) FetchReflectionByID(ctx context.Context, id int64) (internal.Reflection, error) {
	return m.fetchByIDFn(ctx, id)
}
func (m *mockStore) InsertReflection(ctx context.Context, r internal.Reflection) (int64, error) {
	return m.insertFn(ctx, r)
}
func (m *mockStore) UpdateReflection(ctx context.Context, id int64, r internal.Reflection) error {
	return m.updateFn(ctx, id, r)
}
func (m *mockStore) DeleteReflection(ctx context.Context, id int64) error {
	return m.deleteFn(ctx, id)
}

// withStore sets the package-level store to ms for the duration of the test.
func withStore(t *testing.T, ms storeInterface) {
	t.Helper()
	prev := store
	store = ms
	t.Cleanup(func() { store = prev })
}

// doRequest is a helper that builds a request, serves it, and returns the recorder.
func doRequest(method, target, body string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rr := httptest.NewRecorder()

	// Route to correct handler based on method + path pattern
	mux := http.NewServeMux()
	mux.HandleFunc("GET /reflections", handleListReflections)
	mux.HandleFunc("POST /reflections", handleCreateReflection)
	mux.HandleFunc("GET /reflections/{id}", handleGetReflection)
	mux.HandleFunc("PUT /reflections/{id}", handleUpdateReflection)
	mux.HandleFunc("DELETE /reflections/{id}", handleDeleteReflection)
	mux.ServeHTTP(rr, req)
	return rr
}

// --- TestParseID ---

func TestParseID(t *testing.T) {
	cases := []struct {
		raw      string
		wantID   int64
		wantOK   bool
		wantCode int
	}{
		{"1", 1, true, 0},
		{"42", 42, true, 0},
		{fmt.Sprintf("%d", math.MaxInt64), math.MaxInt64, true, 0},
		{"0", 0, false, 400},
		{"-1", 0, false, 400},
		{"abc", 0, false, 400},
		{"", 0, false, 400},
		{"1.5", 0, false, 400},
		{"9223372036854775808", 0, false, 400}, // overflow
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/reflections/"+tc.raw, nil)
			req.SetPathValue("id", tc.raw)
			rr := httptest.NewRecorder()
			id, ok := parseID(rr, req)
			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if ok && id != tc.wantID {
				t.Errorf("id: got %d, want %d", id, tc.wantID)
			}
			if !ok && rr.Code != tc.wantCode {
				t.Errorf("status: got %d, want %d", rr.Code, tc.wantCode)
			}
		})
	}
}

// --- TestParseQueryDate ---

func TestParseQueryDate(t *testing.T) {
	cases := []struct {
		input   string
		wantNil bool // true = expect 0
	}{
		{"", true},
		{"2026-03-29", false},
		{"29-03-2026", true},
		{"2026/03/29", true},
		{"2026-02-30", true},  // invalid date
		{"2024-02-29", false}, // leap year
		{"2023-02-29", true},  // not leap year
		{"0000-01-01", false}, // pre-epoch, valid date → negative unix timestamp
		{"not-a-date", true},
		{" 2026-03-29", true}, // leading space
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := parseQueryDate(tc.input)
			if tc.wantNil && got != 0 {
				t.Errorf("expected 0, got %d", got)
			}
			if !tc.wantNil && got == 0 {
				t.Error("expected non-zero timestamp")
			}
		})
	}

	// Verify the returned value is midnight UTC
	ts := parseQueryDate("2026-03-29")
	expected := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC).Unix()
	if ts != expected {
		t.Errorf("expected midnight UTC %d, got %d", expected, ts)
	}
}

// --- TestHandleListReflections ---

func TestHandleListReflections_ReturnsAll(t *testing.T) {
	ms := &mockStore{
		fetchAllFn: func(_ context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
			return []internal.ReflectionHeader{
				{Id: 1, Title: "A", Tags: []string{"x"}},
				{Id: 2, Title: "B", Tags: []string{"y"}},
			}, nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var headers []internal.ReflectionHeader
	json.NewDecoder(rr.Body).Decode(&headers)
	if len(headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(headers))
	}
}

func TestHandleListReflections_EmptyReturnsArray(t *testing.T) {
	ms := &mockStore{
		fetchAllFn: func(_ context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
			return nil, nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := strings.TrimSpace(rr.Body.String())
	if !strings.HasPrefix(body, "[") {
		t.Errorf("expected JSON array, got: %s", body)
	}
}

func TestHandleListReflections_TitleFilter(t *testing.T) {
	ms := &mockStore{
		fetchAllFn: func(_ context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
			return []internal.ReflectionHeader{
				{Id: 1, Title: "Morning"},
				{Id: 2, Title: "Evening"},
			}, nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections?title=morn", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var headers []internal.ReflectionHeader
	json.NewDecoder(rr.Body).Decode(&headers)
	if len(headers) != 1 || headers[0].Title != "Morning" {
		t.Errorf("expected fuzzy filter to return only 'Morning', got %+v", headers)
	}
}

func TestHandleListReflections_TagsFilter(t *testing.T) {
	ms := &mockStore{
		fetchAllFn: func(_ context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
			return []internal.ReflectionHeader{
				{Id: 1, Title: "A", Tags: []string{"work"}},
				{Id: 2, Title: "B", Tags: []string{"personal"}},
			}, nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections?tags=work", "")
	var headers []internal.ReflectionHeader
	json.NewDecoder(rr.Body).Decode(&headers)
	if len(headers) != 1 || headers[0].Id != 1 {
		t.Errorf("expected only 'work' header, got %+v", headers)
	}
}

func TestHandleListReflections_CreatedAfterPassedToStore(t *testing.T) {
	var capturedFilters internal.FilterOptions
	ms := &mockStore{
		fetchAllFn: func(_ context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
			capturedFilters = f
			return []internal.ReflectionHeader{}, nil
		},
	}
	withStore(t, ms)
	doRequest(http.MethodGet, "/reflections?created_after=2026-01-01", "")
	expected := parseQueryDate("2026-01-01")
	if capturedFilters.CreatedAfter != expected {
		t.Errorf("CreatedAfter not passed to store: got %d, want %d", capturedFilters.CreatedAfter, expected)
	}
}

func TestHandleListReflections_StoreError(t *testing.T) {
	ms := &mockStore{
		fetchAllFn: func(_ context.Context, f internal.FilterOptions) ([]internal.ReflectionHeader, error) {
			return nil, errors.New("db error")
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections", "")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// --- TestHandleGetReflection ---

func TestHandleGetReflection_Valid(t *testing.T) {
	ms := &mockStore{
		fetchByIDFn: func(_ context.Context, id int64) (internal.Reflection, error) {
			return internal.Reflection{
				ReflectionHeader: internal.ReflectionHeader{Id: 1, Title: "T", Tags: []string{}},
				Body:             "body content",
			}, nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections/1", "")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var r internal.Reflection
	json.NewDecoder(rr.Body).Decode(&r)
	if r.Body != "body content" {
		t.Errorf("expected body in response, got %q", r.Body)
	}
}

func TestHandleGetReflection_InvalidID(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodGet, "/reflections/abc", "")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleGetReflection_NotFound(t *testing.T) {
	ms := &mockStore{
		fetchByIDFn: func(_ context.Context, id int64) (internal.Reflection, error) {
			return internal.Reflection{}, pgx.ErrNoRows
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections/999", "")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestHandleGetReflection_StoreError(t *testing.T) {
	ms := &mockStore{
		fetchByIDFn: func(_ context.Context, id int64) (internal.Reflection, error) {
			return internal.Reflection{}, errors.New("db error")
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodGet, "/reflections/1", "")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// --- TestHandleCreateReflection ---

func TestHandleCreateReflection_Valid(t *testing.T) {
	ms := &mockStore{
		insertFn: func(_ context.Context, r internal.Reflection) (int64, error) {
			return 42, nil
		},
	}
	withStore(t, ms)
	body := `{"title":"T","tags":["a"],"body":"B"}`
	rr := doRequest(http.MethodPost, "/reflections", body)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var r internal.Reflection
	json.NewDecoder(rr.Body).Decode(&r)
	if r.Id != 42 {
		t.Errorf("expected id=42, got %d", r.Id)
	}
	if r.CreatedAt == 0 {
		t.Error("expected CreatedAt to be set")
	}
}

func TestHandleCreateReflection_MissingTitle(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodPost, "/reflections", `{"body":"B"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateReflection_MissingBody(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodPost, "/reflections", `{"title":"T"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateReflection_NilTagsCoercedToEmpty(t *testing.T) {
	ms := &mockStore{
		insertFn: func(_ context.Context, r internal.Reflection) (int64, error) {
			return 1, nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodPost, "/reflections", `{"title":"T","body":"B"}`)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	var r internal.Reflection
	json.NewDecoder(rr.Body).Decode(&r)
	if r.Tags == nil {
		t.Error("expected Tags to be non-nil (empty array), got nil")
	}
}

func TestHandleCreateReflection_InvalidJSON(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodPost, "/reflections", `{not json}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateReflection_StoreError(t *testing.T) {
	ms := &mockStore{
		insertFn: func(_ context.Context, r internal.Reflection) (int64, error) {
			return 0, errors.New("db error")
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodPost, "/reflections", `{"title":"T","body":"B"}`)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// --- TestHandleUpdateReflection ---

func TestHandleUpdateReflection_Valid(t *testing.T) {
	ms := &mockStore{
		updateFn: func(_ context.Context, id int64, r internal.Reflection) error {
			return nil
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodPut, "/reflections/1", `{"title":"New","body":"New body"}`)
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestHandleUpdateReflection_InvalidID(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodPut, "/reflections/abc", `{"title":"T","body":"B"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateReflection_MissingTitle(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodPut, "/reflections/1", `{"body":"B"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateReflection_MissingBody(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodPut, "/reflections/1", `{"title":"T"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleUpdateReflection_NotFound(t *testing.T) {
	ms := &mockStore{
		updateFn: func(_ context.Context, id int64, r internal.Reflection) error {
			return pgx.ErrNoRows
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodPut, "/reflections/999", `{"title":"T","body":"B"}`)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestHandleUpdateReflection_StoreError(t *testing.T) {
	ms := &mockStore{
		updateFn: func(_ context.Context, id int64, r internal.Reflection) error {
			return errors.New("db error")
		},
	}
	withStore(t, ms)
	rr := doRequest(http.MethodPut, "/reflections/1", `{"title":"T","body":"B"}`)
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// --- TestHandleDeleteReflection ---

func TestHandleDeleteReflection_Valid(t *testing.T) {
	ms := &mockStore{
		deleteFn: func(_ context.Context, id int64) error { return nil },
	}
	withStore(t, ms)
	rr := doRequest(http.MethodDelete, "/reflections/1", "")
	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", rr.Code)
	}
}

func TestHandleDeleteReflection_InvalidID(t *testing.T) {
	withStore(t, &mockStore{})
	rr := doRequest(http.MethodDelete, "/reflections/abc", "")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestHandleDeleteReflection_NotFound(t *testing.T) {
	ms := &mockStore{
		deleteFn: func(_ context.Context, id int64) error { return pgx.ErrNoRows },
	}
	withStore(t, ms)
	rr := doRequest(http.MethodDelete, "/reflections/999", "")
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestHandleDeleteReflection_StoreError(t *testing.T) {
	ms := &mockStore{
		deleteFn: func(_ context.Context, id int64) error { return errors.New("db error") },
	}
	withStore(t, ms)
	rr := doRequest(http.MethodDelete, "/reflections/1", "")
	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

// --- Fuzz ---

func FuzzParseQueryDate(f *testing.F) {
	f.Add("2026-03-29")
	f.Add("")
	f.Add("not-a-date")
	f.Add("9999-12-31")
	f.Add("0000-01-01")

	f.Fuzz(func(t *testing.T, s string) {
		// Must not panic regardless of input. Result is 0 for invalid/empty,
		// or any int64 Unix timestamp (including negative for pre-epoch dates).
		_ = parseQueryDate(s)
	})
}
