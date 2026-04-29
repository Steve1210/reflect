package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"reflections/internal"

	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if apiURL != "" {
			return fmt.Errorf("--api-url cannot be used with the serve command")
		}

		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}

		http.HandleFunc("GET /reflections", handleListReflections)
		http.HandleFunc("POST /reflections", handleCreateReflection)
		http.HandleFunc("GET /reflections/{id}", handleGetReflection)
		http.HandleFunc("PUT /reflections/{id}", handleUpdateReflection)
		http.HandleFunc("DELETE /reflections/{id}", handleDeleteReflection)

		addr := fmt.Sprintf(":%d", port)
		log.Printf("listening on %s", addr)
		return http.ListenAndServe(addr, nil)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
}

func handleListReflections(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filters := internal.FilterOptions{
		CreatedAfter:  parseQueryDate(q.Get("created_after")),
		CreatedBefore: parseQueryDate(q.Get("created_before")),
		UpdatedAfter:  parseQueryDate(q.Get("updated_after")),
		UpdatedBefore: parseQueryDate(q.Get("updated_before")),
	}

	headers, err := store.FetchAllMetadataWithFilters(context.Background(), filters)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("fetch reflections: %v", err)
		return
	}
	if headers == nil {
		headers = []internal.ReflectionHeader{}
	}

	if title := q.Get("title"); title != "" {
		headers = internal.FilterByTitle(headers, title)
	}
	if tags := q.Get("tags"); tags != "" {
		headers = internal.FilterByTags(headers, strings.Split(tags, ","))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(headers)
}

func handleGetReflection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	reflection, err := store.FetchReflectionByID(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("fetch reflection %d: %v", id, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reflection)
}

func handleCreateReflection(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title string   `json:"title"`
		Tags  []string `json:"tags"`
		Body  string   `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if body.Body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}
	if body.Tags == nil {
		body.Tags = []string{}
	}

	now := time.Now().Unix()
	reflection := internal.Reflection{
		ReflectionHeader: internal.ReflectionHeader{
			Title:     body.Title,
			Tags:      body.Tags,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Body: body.Body,
	}

	id, err := store.InsertReflection(context.Background(), reflection)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("insert reflection: %v", err)
		return
	}

	reflection.Id = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reflection)
}

func handleUpdateReflection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var body struct {
		Title string   `json:"title"`
		Tags  []string `json:"tags"`
		Body  string   `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Title == "" {
		http.Error(w, "title is required", http.StatusBadRequest)
		return
	}
	if body.Body == "" {
		http.Error(w, "body is required", http.StatusBadRequest)
		return
	}
	if body.Tags == nil {
		body.Tags = []string{}
	}

	reflection := internal.Reflection{
		ReflectionHeader: internal.ReflectionHeader{
			Title:     body.Title,
			Tags:      body.Tags,
			UpdatedAt: time.Now().Unix(),
		},
		Body: body.Body,
	}

	err := store.UpdateReflection(context.Background(), id, reflection)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("update reflection %d: %v", id, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteReflection(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	err := store.DeleteReflection(context.Background(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		log.Printf("delete reflection %d: %v", id, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// parseID extracts and validates the {id} path value. Returns false and writes an error response if invalid.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// parseQueryDate converts a YYYY-MM-DD query param to a Unix timestamp. Returns 0 if empty or invalid.
func parseQueryDate(s string) int64 {
	if s == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return 0
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix()
}
