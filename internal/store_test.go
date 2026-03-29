package internal

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var testDB *pgxpool.Pool
var testStore *Store

func TestMain(m *testing.M) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://steve@localhost:5432/steve"
	}

	ctx := context.Background()
	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		panic("failed to connect to test DB: " + err.Error())
	}
	defer testDB.Close()

	testStore = NewStore(testDB)

	truncate(ctx)
	code := m.Run()
	truncate(ctx)

	os.Exit(code)
}

func truncate(ctx context.Context) {
	_, err := testDB.Exec(ctx, "TRUNCATE reflections RESTART IDENTITY")
	if err != nil {
		panic("failed to truncate: " + err.Error())
	}
}

func cleanup(t *testing.T) {
	t.Helper()
	truncate(context.Background())
	t.Cleanup(func() { truncate(context.Background()) })
}

func makeReflection(title, body string, tags []string) Reflection {
	now := time.Now().Unix()
	return Reflection{
		ReflectionHeader: ReflectionHeader{
			Title:     title,
			Tags:      tags,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Body: body,
	}
}

// --- InsertReflection ---

func TestInsertReflection_Valid(t *testing.T) {
	cleanup(t)
	r := makeReflection("Test title", "Test body", []string{"a", "b"})
	id, err := testStore.InsertReflection(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive ID, got %d", id)
	}
}

func TestInsertReflection_IDIncrements(t *testing.T) {
	cleanup(t)
	r := makeReflection("First", "body", []string{})
	id1, _ := testStore.InsertReflection(context.Background(), r)
	id2, _ := testStore.InsertReflection(context.Background(), makeReflection("Second", "body", []string{}))
	if id2 <= id1 {
		t.Errorf("expected id2 (%d) > id1 (%d)", id2, id1)
	}
}

func TestInsertReflection_EmptyTagsAllowed(t *testing.T) {
	cleanup(t)
	r := makeReflection("Title", "body", []string{})
	_, err := testStore.InsertReflection(context.Background(), r)
	if err != nil {
		t.Fatalf("unexpected error with empty tags: %v", err)
	}
}

// --- FetchReflectionByID ---

func TestFetchReflectionByID_Valid(t *testing.T) {
	cleanup(t)
	r := makeReflection("My title", "My body", []string{"x", "y"})
	id, _ := testStore.InsertReflection(context.Background(), r)

	got, err := testStore.FetchReflectionByID(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Title != r.Title {
		t.Errorf("title mismatch: got %q, want %q", got.Title, r.Title)
	}
	if got.Body != r.Body {
		t.Errorf("body mismatch: got %q, want %q", got.Body, r.Body)
	}
	if len(got.Tags) != 2 {
		t.Errorf("expected 2 tags, got %v", got.Tags)
	}
	if got.CreatedAt != r.CreatedAt {
		t.Errorf("created_at mismatch: got %d, want %d", got.CreatedAt, r.CreatedAt)
	}
}

func TestFetchReflectionByID_NotFound(t *testing.T) {
	cleanup(t)
	_, err := testStore.FetchReflectionByID(context.Background(), 99999)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestFetchReflectionByID_ZeroID(t *testing.T) {
	cleanup(t)
	_, err := testStore.FetchReflectionByID(context.Background(), 0)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows for id=0, got %v", err)
	}
}

// --- FetchAllMetadata ---

func TestFetchAllMetadata_Empty(t *testing.T) {
	cleanup(t)
	headers, err := testStore.FetchAllMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected empty slice, got %d rows", len(headers))
	}
}

func TestFetchAllMetadata_OrderedByCreatedAtDesc(t *testing.T) {
	cleanup(t)
	now := time.Now().Unix()
	r1 := Reflection{ReflectionHeader: ReflectionHeader{Title: "Older", Tags: []string{}, CreatedAt: now - 100, UpdatedAt: now - 100}, Body: "b"}
	r2 := Reflection{ReflectionHeader: ReflectionHeader{Title: "Newer", Tags: []string{}, CreatedAt: now, UpdatedAt: now}, Body: "b"}

	testStore.InsertReflection(context.Background(), r1)
	testStore.InsertReflection(context.Background(), r2)

	headers, err := testStore.FetchAllMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 2 {
		t.Fatalf("expected 2, got %d", len(headers))
	}
	if headers[0].Title != "Newer" {
		t.Errorf("expected 'Newer' first, got %q", headers[0].Title)
	}
}

// --- FetchAllMetadataWithFilters ---

func TestFetchAllMetadataWithFilters_NoFilters(t *testing.T) {
	cleanup(t)
	testStore.InsertReflection(context.Background(), makeReflection("A", "b", []string{}))
	testStore.InsertReflection(context.Background(), makeReflection("B", "b", []string{}))

	headers, err := testStore.FetchAllMetadataWithFilters(context.Background(), FilterOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 2 {
		t.Errorf("expected 2, got %d", len(headers))
	}
}

func TestFetchAllMetadataWithFilters_CreatedAfter(t *testing.T) {
	cleanup(t)
	now := time.Now().Unix()
	old := Reflection{ReflectionHeader: ReflectionHeader{Title: "Old", Tags: []string{}, CreatedAt: now - 1000, UpdatedAt: now - 1000}, Body: "b"}
	recent := Reflection{ReflectionHeader: ReflectionHeader{Title: "Recent", Tags: []string{}, CreatedAt: now, UpdatedAt: now}, Body: "b"}
	testStore.InsertReflection(context.Background(), old)
	testStore.InsertReflection(context.Background(), recent)

	headers, err := testStore.FetchAllMetadataWithFilters(context.Background(), FilterOptions{CreatedAfter: now - 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 1 || headers[0].Title != "Recent" {
		t.Errorf("expected only 'Recent', got %+v", headers)
	}
}

func TestFetchAllMetadataWithFilters_CreatedBefore(t *testing.T) {
	cleanup(t)
	now := time.Now().Unix()
	old := Reflection{ReflectionHeader: ReflectionHeader{Title: "Old", Tags: []string{}, CreatedAt: now - 1000, UpdatedAt: now - 1000}, Body: "b"}
	recent := Reflection{ReflectionHeader: ReflectionHeader{Title: "Recent", Tags: []string{}, CreatedAt: now, UpdatedAt: now}, Body: "b"}
	testStore.InsertReflection(context.Background(), old)
	testStore.InsertReflection(context.Background(), recent)

	headers, err := testStore.FetchAllMetadataWithFilters(context.Background(), FilterOptions{CreatedBefore: now - 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 1 || headers[0].Title != "Old" {
		t.Errorf("expected only 'Old', got %+v", headers)
	}
}

func TestFetchAllMetadataWithFilters_ConflictingFilters(t *testing.T) {
	cleanup(t)
	now := time.Now().Unix()
	testStore.InsertReflection(context.Background(), makeReflection("A", "b", []string{}))

	headers, err := testStore.FetchAllMetadataWithFilters(context.Background(), FilterOptions{
		CreatedAfter:  now + 1000,
		CreatedBefore: now - 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 0 {
		t.Errorf("expected no results for conflicting filters, got %d", len(headers))
	}
}

func TestFetchAllMetadataWithFilters_UpdatedAfter(t *testing.T) {
	cleanup(t)
	now := time.Now().Unix()
	r1 := Reflection{ReflectionHeader: ReflectionHeader{Title: "A", Tags: []string{}, CreatedAt: now, UpdatedAt: now - 1000}, Body: "b"}
	r2 := Reflection{ReflectionHeader: ReflectionHeader{Title: "B", Tags: []string{}, CreatedAt: now, UpdatedAt: now}, Body: "b"}
	testStore.InsertReflection(context.Background(), r1)
	testStore.InsertReflection(context.Background(), r2)

	headers, err := testStore.FetchAllMetadataWithFilters(context.Background(), FilterOptions{UpdatedAfter: now - 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 1 || headers[0].Title != "B" {
		t.Errorf("expected only 'B', got %+v", headers)
	}
}

// --- UpdateReflection ---

func TestUpdateReflection_Valid(t *testing.T) {
	cleanup(t)
	r := makeReflection("Original", "Original body", []string{"old"})
	id, _ := testStore.InsertReflection(context.Background(), r)

	originalCreatedAt := r.CreatedAt
	updated := Reflection{
		ReflectionHeader: ReflectionHeader{
			Title:     "Updated",
			Tags:      []string{"new"},
			UpdatedAt: originalCreatedAt + 60, // explicitly different from created_at
		},
		Body: "Updated body",
	}

	err := testStore.UpdateReflection(context.Background(), id, updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := testStore.FetchReflectionByID(context.Background(), id)
	if got.Title != "Updated" {
		t.Errorf("expected updated title, got %q", got.Title)
	}
	if got.Body != "Updated body" {
		t.Errorf("expected updated body, got %q", got.Body)
	}
	if got.CreatedAt != originalCreatedAt {
		t.Errorf("created_at should not change: was %d, got %d", originalCreatedAt, got.CreatedAt)
	}
	if got.UpdatedAt == originalCreatedAt {
		t.Error("updated_at should have changed")
	}
}

func TestUpdateReflection_NotFound(t *testing.T) {
	cleanup(t)
	updated := makeReflection("X", "body", []string{})
	err := testStore.UpdateReflection(context.Background(), 99999, updated)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}
}

// --- DeleteReflection ---

func TestDeleteReflection_Valid(t *testing.T) {
	cleanup(t)
	r := makeReflection("To delete", "body", []string{})
	id, _ := testStore.InsertReflection(context.Background(), r)

	err := testStore.DeleteReflection(context.Background(), id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = testStore.FetchReflectionByID(context.Background(), id)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows after delete, got %v", err)
	}
}

func TestDeleteReflection_NotFound(t *testing.T) {
	cleanup(t)
	err := testStore.DeleteReflection(context.Background(), 99999)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestDeleteReflection_AlreadyDeleted(t *testing.T) {
	cleanup(t)
	r := makeReflection("To delete twice", "body", []string{})
	id, _ := testStore.InsertReflection(context.Background(), r)
	testStore.DeleteReflection(context.Background(), id)

	err := testStore.DeleteReflection(context.Background(), id)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows on second delete, got %v", err)
	}
}
