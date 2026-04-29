package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"reflections/internal"

	"github.com/spf13/cobra"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View the full content of a reflection",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		titleFilter, _ := cmd.Flags().GetString("title")
		tagsFilter, _ := cmd.Flags().GetStringSlice("tags")

		headers, err := store.FetchAllMetadataWithFilters(ctx, internal.FilterOptions{})
		if err != nil {
			return fmt.Errorf("fetch reflections: %w", err)
		}
		if titleFilter != "" {
			headers = internal.FilterByTitle(headers, titleFilter)
		}
		if len(tagsFilter) > 0 {
			headers = internal.FilterByTags(headers, tagsFilter)
		}
		if len(headers) == 0 {
			fmt.Println("No reflections found.")
			return nil
		}

		for i, h := range headers {
			fmt.Printf("%d. %s (Tags: %v)\n", i+1, h.Title, h.Tags)
		}

		fmt.Print("\nEnter number: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(headers) {
			return fmt.Errorf("invalid selection")
		}

		reflection, err := store.FetchReflectionByID(ctx, headers[n-1].Id)
		if err != nil {
			return fmt.Errorf("fetch reflection: %w", err)
		}

		fmt.Printf("\nTitle: %s\n", reflection.Title)
		fmt.Printf("Tags:  %v\n", reflection.Tags)
		fmt.Printf("Created: %s\n", time.Unix(reflection.CreatedAt, 0).Format("2006-01-02 15:04"))
		fmt.Printf("Updated: %s\n", time.Unix(reflection.UpdatedAt, 0).Format("2006-01-02 15:04"))
		fmt.Println("---")
		fmt.Println(reflection.Body)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
	viewCmd.Flags().StringP("title", "t", "", "Filter reflections by title (fuzzy match)")
	viewCmd.Flags().StringSliceP("tags", "g", nil, "Filter reflections by tags (comma-separated)")
}
