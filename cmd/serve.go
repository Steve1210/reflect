package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflections/internal"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, err := cmd.Flags().GetInt("port")
		if err != nil {
			return err
		}

		http.HandleFunc("GET /reflections", func(w http.ResponseWriter, r *http.Request) {
			headers, err := store.FetchAllMetadataWithFilters(context.Background(), internal.FilterOptions{})
			if err != nil {
				http.Error(w, "internal server error", http.StatusInternalServerError)
				log.Printf("fetch reflections: %v", err)
				return
			}
			if headers == nil {
				headers = []internal.ReflectionHeader{}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(headers)
		})

		addr := fmt.Sprintf(":%d", port)
		log.Printf("listening on %s", addr)
		return http.ListenAndServe(addr, nil)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
}
