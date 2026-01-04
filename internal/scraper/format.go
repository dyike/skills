package scraper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dyike/skills/models/scraper"
)

// OutputFormat represents the output format type
type OutputFormat string

const (
	FormatText     OutputFormat = "text"
	FormatMarkdown OutputFormat = "markdown"
	FormatJSON     OutputFormat = "json"
)

// FormatOutput formats scraped items for output
func FormatOutput(items []scraper.NewsItem, format OutputFormat) (string, error) {
	switch format {
	case FormatJSON:
		return formatJSON(items)
	case FormatMarkdown:
		return formatMarkdown(items), nil
	default:
		return formatText(items), nil
	}
}

func formatJSON(items []scraper.NewsItem) (string, error) {
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatMarkdown(items []scraper.NewsItem) string {
	var lines []string
	var currentSource string

	for _, item := range items {
		if item.Source != currentSource {
			if currentSource != "" {
				lines = append(lines, "")
			}
			lines = append(lines, fmt.Sprintf("## %s", strings.ToUpper(item.Source)))
			lines = append(lines, "")
			currentSource = item.Source
		}

		var scoreInfo, commentsInfo string
		if item.Score != nil {
			scoreInfo = fmt.Sprintf(" (%d pts)", *item.Score)
		}
		if item.Comments != nil {
			commentsInfo = fmt.Sprintf(" | %d comments", *item.Comments)
		}

		lines = append(lines, fmt.Sprintf("- [%s](%s)%s%s", item.Title, item.URL, scoreInfo, commentsInfo))

		if item.Tagline != "" {
			lines = append(lines, fmt.Sprintf("  > %s", item.Tagline))
		}
	}

	return strings.Join(lines, "\n")
}

func formatText(items []scraper.NewsItem) string {
	var lines []string
	var currentSource string
	idx := 1

	for _, item := range items {
		if item.Source != currentSource {
			if currentSource != "" {
				lines = append(lines, "")
			}
			lines = append(lines, fmt.Sprintf("=== %s ===", strings.ToUpper(item.Source)))
			currentSource = item.Source
		}

		var scoreInfo, commentsInfo string
		if item.Score != nil {
			scoreInfo = fmt.Sprintf(" [%d pts]", *item.Score)
		}
		if item.Comments != nil {
			commentsInfo = fmt.Sprintf(" [%d comments]", *item.Comments)
		}

		lines = append(lines, fmt.Sprintf("%d. %s%s%s", idx, item.Title, scoreInfo, commentsInfo))
		lines = append(lines, fmt.Sprintf("   %s", item.URL))

		if item.Tagline != "" {
			lines = append(lines, fmt.Sprintf("   → %s", item.Tagline))
		}

		idx++
	}

	return strings.Join(lines, "\n")
}
