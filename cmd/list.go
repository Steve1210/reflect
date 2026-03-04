package cmd

import (
	"context"
	"fmt"
	"reflections/internal"
	"time"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List your reflections",
	Long:  `List your reflections. This command will display the titles and tags of all saved reflections.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		title, err := cmd.Flags().GetString("title")
		if err != nil {
			return fmt.Errorf("parse title flag: %w", err)
		}
		tags, err := cmd.Flags().GetStringSlice("tags")
		if err != nil {
			return fmt.Errorf("parse tags flag: %w", err)
		}

		createdAfter, err := parseDate(cmd, "created-after")
		if err != nil {
			return err
		}
		createdBefore, err := parseDate(cmd, "created-before")
		if err != nil {
			return err
		}
		updatedAfter, err := parseDate(cmd, "updated-after")
		if err != nil {
			return err
		}
		updatedBefore, err := parseDate(cmd, "updated-before")
		if err != nil {
			return err
		}

		filters := internal.FilterOptions{
			CreatedAfter:  createdAfter,
			CreatedBefore: createdBefore,
			UpdatedAfter:  updatedAfter,
			UpdatedBefore: updatedBefore,
		}

		ctx := context.Background()

		results, err := store.FetchAllMetadataWithFilters(ctx, filters)
		if err != nil {
			return fmt.Errorf("load reflections: %w", err)
		}

		results = internal.FilterByTitle(results, title)
		results = internal.FilterByTags(results, tags)

		if len(results) == 0 {
			fmt.Println("No reflections found.")
			return nil
		}

		fmt.Println("Your Reflections:")
		for _, r := range results {
			fmt.Printf("- %s (Tags: %v)\n", r.Title, r.Tags)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().StringP("title", "t", "", "Search by title (fuzzy)")
	listCmd.Flags().StringSliceP("tags", "g", []string{}, "Filter by tags")
	listCmd.Flags().String("created-after", "", "Only show reflections created on or after this date (YYYY-MM-DD)")
	listCmd.Flags().String("created-before", "", "Only show reflections created before this date (YYYY-MM-DD)")
	listCmd.Flags().String("updated-after", "", "Only show reflections updated on or after this date (YYYY-MM-DD)")
	listCmd.Flags().String("updated-before", "", "Only show reflections updated before this date (YYYY-MM-DD)")
}

func parseDate(cmd *cobra.Command, flag string) (int64, error) {
	s, err := cmd.Flags().GetString(flag)
	if err != nil {
		return 0, fmt.Errorf("parse %s flag: %w", flag, err)
	}
	if s == "" {
		return 0, nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return 0, fmt.Errorf("invalid date for --%s %q, expected YYYY-MM-DD", flag, s)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC).Unix(), nil
}
