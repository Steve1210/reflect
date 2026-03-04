package cmd

import (
	"bytes"
	"context"
	"fmt"
	"reflections/internal"
	"strings"

	"github.com/spf13/cobra"
)

var (
	titleFlag string
	tagsFlag  []string
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new reflection",
	Long:  `Create a new reflection. You can specify the title and tags for the reflection.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		template := buildTemplate(titleFlag, tagsFlag)
		ctx := context.Background()

		content, err := internal.OpenInEditor(template)
		if err != nil {
			return fmt.Errorf("open editor: %w", err)
		}

		reflection, err := internal.ParseContent(content)
		if err != nil {
			return fmt.Errorf("parse reflection content: %w", err)
		}

		fmt.Printf("Created reflection:\nTitle: %s\nTags: %v\nBody:\n%s\n", reflection.Title, reflection.Tags, reflection.Body)

		// // OLD Write to file in the reflections directory
		//filepath, err := internal.SaveToFile(reflection)

		// if err != nil {
		// 	return fmt.Errorf("save reflection to file: %w", err)
		// }

		// fmt.Printf("Reflection saved successfully to %s\n", filepath)

		// NEW Write the reflection into the db
		id, err := store.InsertReflection(ctx, reflection)

		if err != nil {
			return fmt.Errorf("insert reflection into database: %w", err)
		}

		fmt.Printf("------\nReflected! (%d)\n------\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	createCmd.Flags().StringVarP(&titleFlag, "title", "t", "", "Title of the reflection")
	createCmd.Flags().StringSliceVarP(&tagsFlag, "tags", "g", []string{}, "Tags for the reflection (comma-separated)")
}

func buildTemplate(title string, tags []string) string {

	var buf bytes.Buffer

	var tagsLine string
	if len(tags) > 0 {
		// Trim leading and trailing whitespace from each tag
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
		tagsLine = strings.Join(tags, ", ")
	}

	buf.WriteString("# Lines starting with # are ignored.\n")
	buf.WriteString("# Reflecting on...\n")
	buf.WriteString("Title: " + title + "\n")
	buf.WriteString("Tags: " + tagsLine + "\n")
	buf.WriteString("---\n")

	return buf.String()

}
