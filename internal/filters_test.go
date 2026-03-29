package internal

import (
	"testing"
)

// helpers

func makeHeaders(titles ...string) []ReflectionHeader {
	headers := make([]ReflectionHeader, len(titles))
	for i, t := range titles {
		headers[i] = ReflectionHeader{Id: int64(i + 1), Title: t}
	}
	return headers
}

// --- FilterByTitle ---

func TestFilterByTitle_EmptyTitle(t *testing.T) {
	headers := makeHeaders("Alpha", "Beta", "Gamma")
	got := FilterByTitle(headers, "")
	if len(got) != len(headers) {
		t.Fatalf("expected all %d headers, got %d", len(headers), len(got))
	}
}

func TestFilterByTitle_EmptyHeaders(t *testing.T) {
	got := FilterByTitle(nil, "alpha")
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %d", len(got))
	}
}

func TestFilterByTitle_Match(t *testing.T) {
	headers := makeHeaders("Morning thoughts", "Evening reflections", "Midday break")
	got := FilterByTitle(headers, "morn")
	if len(got) == 0 {
		t.Fatal("expected at least one fuzzy match for 'morn'")
	}
	if got[0].Title != "Morning thoughts" {
		t.Errorf("expected 'Morning thoughts', got %q", got[0].Title)
	}
}

func TestFilterByTitle_NoMatch(t *testing.T) {
	headers := makeHeaders("Alpha", "Beta")
	got := FilterByTitle(headers, "zzzzznomatch")
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}

// --- FilterByTags ---

func TestFilterByTags_EmptyTags(t *testing.T) {
	headers := makeHeaders("A", "B")
	headers[0].Tags = []string{"work"}
	got := FilterByTags(headers, []string{})
	if len(got) != len(headers) {
		t.Fatalf("expected all %d headers, got %d", len(headers), len(got))
	}
}

func TestFilterByTags_EmptyHeaders(t *testing.T) {
	got := FilterByTags(nil, []string{"work"})
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

func TestFilterByTags_CaseInsensitive(t *testing.T) {
	headers := []ReflectionHeader{
		{Id: 1, Title: "A", Tags: []string{"TODO"}},
		{Id: 2, Title: "B", Tags: []string{"done"}},
	}
	got := FilterByTags(headers, []string{"todo"})
	if len(got) != 1 || got[0].Id != 1 {
		t.Errorf("expected header 1 (case-insensitive match), got %+v", got)
	}
}

func TestFilterByTags_ORLogic(t *testing.T) {
	headers := []ReflectionHeader{
		{Id: 1, Title: "A", Tags: []string{"work"}},
		{Id: 2, Title: "B", Tags: []string{"personal"}},
		{Id: 3, Title: "C", Tags: []string{"other"}},
	}
	got := FilterByTags(headers, []string{"work", "personal"})
	if len(got) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got))
	}
}

func TestFilterByTags_NoTagsOnHeader(t *testing.T) {
	headers := []ReflectionHeader{
		{Id: 1, Title: "A", Tags: []string{}},
	}
	got := FilterByTags(headers, []string{"work"})
	if len(got) != 0 {
		t.Fatalf("expected no matches for header with empty tags, got %d", len(got))
	}
}

func TestFilterByTags_NoMatch(t *testing.T) {
	headers := []ReflectionHeader{
		{Id: 1, Title: "A", Tags: []string{"work"}},
	}
	got := FilterByTags(headers, []string{"personal"})
	if len(got) != 0 {
		t.Fatalf("expected no matches, got %d", len(got))
	}
}

func TestFilterByTags_NilTagsSlice(t *testing.T) {
	headers := makeHeaders("A", "B")
	got := FilterByTags(headers, nil)
	if len(got) != len(headers) {
		t.Fatalf("expected all headers for nil tags, got %d", len(got))
	}
}

// --- Fuzz ---

func FuzzFilterByTitle(f *testing.F) {
	f.Add("morning")
	f.Add("")
	f.Add("!@#$%")
	f.Add("aaaaaaaaaaaaaaaaaaaaaaaaa")

	headers := makeHeaders("Morning thoughts", "Evening notes", "Daily reflection")
	f.Fuzz(func(t *testing.T, title string) {
		result := FilterByTitle(headers, title)
		// must not panic; result must be a subset of original
		for _, r := range result {
			found := false
			for _, h := range headers {
				if h.Id == r.Id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("result contains header not in input: %+v", r)
			}
		}
	})
}

func FuzzFilterByTags(f *testing.F) {
	f.Add("work")
	f.Add("")
	f.Add("WORK")
	f.Add("tag with spaces")

	headers := []ReflectionHeader{
		{Id: 1, Title: "A", Tags: []string{"work", "daily"}},
		{Id: 2, Title: "B", Tags: []string{"personal"}},
		{Id: 3, Title: "C", Tags: []string{}},
	}
	f.Fuzz(func(t *testing.T, tag string) {
		result := FilterByTags(headers, []string{tag})
		for _, r := range result {
			found := false
			for _, h := range headers {
				if h.Id == r.Id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("result contains header not in input: %+v", r)
			}
		}
	})
}
