package internal

import (
	"strings"

	"github.com/sahilm/fuzzy"
)

func FilterByTitle(headers []ReflectionHeader, title string) []ReflectionHeader {
	if title == "" {
		return headers
	}
	titles := make([]string, len(headers))
	for i, h := range headers {
		titles[i] = h.Title
	}
	matches := fuzzy.Find(title, titles)
	out := make([]ReflectionHeader, len(matches))
	for i, m := range matches {
		out[i] = headers[m.Index]
	}
	return out
}

func FilterByTags(headers []ReflectionHeader, tags []string) []ReflectionHeader {
	if len(tags) == 0 {
		return headers
	}
	var out []ReflectionHeader
	for _, h := range headers {
		for _, filterTag := range tags {
			for _, t := range h.Tags {
				if strings.EqualFold(t, filterTag) {
					out = append(out, h)
					goto next
				}
			}
		}
	next:
	}
	return out
}
