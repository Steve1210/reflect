package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- ParseContent ---

func TestParseContent_Valid(t *testing.T) {
	input := "Title: My Reflection\nTags: work, daily\n---\nThis is the body."
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Title != "My Reflection" {
		t.Errorf("expected title 'My Reflection', got %q", r.Title)
	}
	if len(r.Tags) != 2 || r.Tags[0] != "work" || r.Tags[1] != "daily" {
		t.Errorf("unexpected tags: %v", r.Tags)
	}
	if r.Body != "This is the body." {
		t.Errorf("unexpected body: %q", r.Body)
	}
	if r.CreatedAt == 0 || r.UpdatedAt == 0 {
		t.Error("expected non-zero timestamps")
	}
}

func TestParseContent_TagsWithSpaces(t *testing.T) {
	input := "Title: T\nTags: tag1 , tag2 , tag3\n---\nbody"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Tags) != 3 {
		t.Fatalf("expected 3 tags, got %v", r.Tags)
	}
	for _, tag := range r.Tags {
		if tag != strings.TrimSpace(tag) {
			t.Errorf("tag not trimmed: %q", tag)
		}
	}
}

func TestParseContent_EmptyTagsLine(t *testing.T) {
	input := "Title: T\nTags:\n---\nbody"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.Tags) != 0 {
		t.Errorf("expected empty tags, got %v", r.Tags)
	}
}

func TestParseContent_TagsWithEmptyEntries(t *testing.T) {
	input := "Title: T\nTags: tag1,,tag2\n---\nbody"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, tag := range r.Tags {
		if tag == "" {
			t.Error("empty string found in tags")
		}
	}
}

func TestParseContent_MissingTitle(t *testing.T) {
	input := "Tags: work\n---\nbody"
	_, err := ParseContent(input)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestParseContent_WhitespaceTitle(t *testing.T) {
	input := "Title:    \nTags:\n---\nbody"
	_, err := ParseContent(input)
	if err == nil {
		t.Fatal("expected error for whitespace-only title")
	}
}

func TestParseContent_MissingBody(t *testing.T) {
	input := "Title: T\nTags:\n---\n"
	_, err := ParseContent(input)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestParseContent_WhitespaceBody(t *testing.T) {
	input := "Title: T\nTags:\n---\n   \n   "
	_, err := ParseContent(input)
	if err == nil {
		t.Fatal("expected error for whitespace-only body")
	}
}

func TestParseContent_NoSeparator(t *testing.T) {
	input := "Title: T\nTags:\nsome body without separator"
	_, err := ParseContent(input)
	// No --- means no body section, so body is empty → error
	if err == nil {
		t.Fatal("expected error when no --- separator")
	}
}

func TestParseContent_CommentLinesSkipped(t *testing.T) {
	input := "# This is a comment\nTitle: T\n# Another comment\nTags:\n---\nbody"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Title != "T" {
		t.Errorf("unexpected title: %q", r.Title)
	}
}

func TestParseContent_BlankLinesInHeaderSkipped(t *testing.T) {
	input := "\n\nTitle: T\n\nTags: work\n---\nbody"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Title != "T" {
		t.Errorf("unexpected title: %q", r.Title)
	}
}

func TestParseContent_MultipleSeparators(t *testing.T) {
	input := "Title: T\nTags:\n---\nfirst part\n---\nsecond part"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(r.Body, "second part") {
		t.Errorf("expected body to contain second part after second ---, got %q", r.Body)
	}
}

func TestParseContent_TimestampsSet(t *testing.T) {
	input := "Title: T\nTags:\n---\nbody"
	r, err := ParseContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.CreatedAt == 0 {
		t.Error("CreatedAt should be non-zero")
	}
	if r.UpdatedAt == 0 {
		t.Error("UpdatedAt should be non-zero")
	}
}

// --- OpenInEditor ---

func TestOpenInEditor_FakeEditor(t *testing.T) {
	// Write a fake editor script that overwrites the given file with known content
	dir := t.TempDir()
	script := filepath.Join(dir, "fake_editor.sh")
	content := "#!/bin/sh\nprintf 'Title: Fake\\nTags: test\\n---\\nFake body.' > \"$1\"\n"
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}

	t.Setenv("EDITOR", script)

	result, err := OpenInEditor("# initial content\nTitle: \nTags: \n---\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Fake body.") {
		t.Errorf("expected fake body in result, got: %q", result)
	}
}

func TestOpenInEditor_NonExistentEditor(t *testing.T) {
	t.Setenv("EDITOR", "/nonexistent/editor_that_does_not_exist")
	_, err := OpenInEditor("some content")
	if err == nil {
		t.Fatal("expected error for non-existent editor")
	}
}

// --- Fuzz ---

func FuzzParseContent(f *testing.F) {
	f.Add("Title: T\nTags: a,b\n---\nbody here")
	f.Add("Title: T\n---\nbody")
	f.Add("")
	f.Add("# comment\nTitle: test\nTags:\n---\nsome body")
	f.Add("Title:   \n---\n   ")

	f.Fuzz(func(t *testing.T, content string) {
		// Must not panic regardless of input
		_, _ = ParseContent(content)
	})
}
