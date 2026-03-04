package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func OpenInEditor(content string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	tmpFile, err := os.CreateTemp("", "reflection_*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = tmpFile.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to open editor: %w", err)
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited content: %w", err)
	}

	return string(edited), nil

}

func ParseContent(content string) (Reflection, error) {
	lines := strings.Split(content, "\n")

	var headerLines []string
	var bodyLines []string
	inBody := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inBody {
			if trimmed == "---" {
				inBody = true
				continue
			}
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				// skip blank lines and comment lines in the header
				continue
			}
			headerLines = append(headerLines, trimmed)
		} else {
			// preserve body lines as-is (including leading/trailing spaces)
			bodyLines = append(bodyLines, line)
		}
	}

	var title string
	var tags []string
	for _, hl := range headerLines {
		if strings.HasPrefix(hl, "Title:") {
			title = strings.TrimSpace(strings.TrimPrefix(hl, "Title:"))
		} else if strings.HasPrefix(hl, "Tags:") {
			tagsLine := strings.TrimSpace(strings.TrimPrefix(hl, "Tags:"))
			if tagsLine != "" {
				parts := strings.Split(tagsLine, ",")
				for _, part := range parts {
					trimmed := strings.TrimSpace(part)
					if trimmed != "" {
						tags = append(tags, trimmed)
					}
				}
			}
		}
	}

	if title == "" {
		return Reflection{}, fmt.Errorf("Title can't be empty!")
	}

	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	if body == "" {
		return Reflection{}, fmt.Errorf("body is empty")
	}

	reflection := Reflection{
		ReflectionHeader: ReflectionHeader{
			Title:     title,
			Tags:      tags,
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
		Body: body,
	}

	return reflection, nil
}

func SaveToFile(reflection Reflection) (string, error) {
	filepath := fmt.Sprintf("reflections/%s.json", strings.ReplaceAll(reflection.Title, " ", "_"))
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(reflection); err != nil {
		return "", fmt.Errorf("failed to encode reflection to JSON: %w", err)
	}

	return filepath, nil
}

func LoadReflectionHeaders() ([]ReflectionHeader, error) {
	files, err := os.ReadDir("reflections")
	if err != nil {
		return nil, fmt.Errorf("failed to read reflections directory: %w", err)
	}

	var headers []ReflectionHeader
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(fmt.Sprintf("reflections/%s", file.Name()))
		if err != nil {
			fmt.Printf("failed to read file %s: %v\n", file.Name(), err)
			continue
		}

		var head ReflectionHeader
		if err := json.Unmarshal(data, &head); err != nil {
			fmt.Printf("failed to unmarshal JSON from file %s: %v\n", file.Name(), err)
			continue
		}

		headers = append(headers, head)
	}

	return headers, nil
}
