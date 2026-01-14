package scraper

import (
	"fmt"
	"os"

	"github.com/crosszan/modu/repos/scraper"
	"github.com/spf13/cobra"
)

// config holds global flags
var cfg struct {
	limit  int
	format string
	output string
}

// NewCmd creates the root scraper command
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "news-scraper",
		Short: "Fetch latest content from HN, Product Hunt, newsletters, and more",
	}

	root.PersistentFlags().IntVarP(&cfg.limit, "limit", "l", 5, "Max items to scrape")
	root.PersistentFlags().StringVarP(&cfg.format, "format", "f", "text", "Output format: text, markdown, json")
	root.PersistentFlags().StringVarP(&cfg.output, "output", "o", "", "Output file path")

	// Simple commands (no extra args)
	addSimple(root, "hn", "Scrape Hacker News", scraper.ScrapeHN)
	addSimple(root, "ph", "Scrape Product Hunt", scraper.ScrapePH)
	addSimple(root, "twitter-trending", "Scrape Twitter/X trending", scraper.ScrapeTwitterTrending)
	addSimple(root, "xhs", "Scrape Xiaohongshu", scraper.ScrapeXHS)

	// Commands with string arg
	addWithArg(root, "twitter-user", "Scrape Twitter/X user timeline", "user", "u", scraper.ScrapeTwitterUser)
	addWithArg(root, "newsletter", "Scrape newsletter archive", "url", "u", func(url string, limit int) ([]scraper.NewsItem, error) {
		return scraper.ScrapeNewsletter(url, nil, limit)
	})

	return root
}

func addSimple(root *cobra.Command, use, short string, fn func(int) ([]scraper.NewsItem, error)) {
	root.AddCommand(&cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Scraping %s...\n", use)
			items, err := fn(cfg.limit)
			if err != nil {
				return err
			}
			return output(items)
		},
	})
}

func addWithArg(root *cobra.Command, use, short, argName, argShort string, fn func(string, int) ([]scraper.NewsItem, error)) {
	var arg string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Scraping %s (%s)...\n", use, arg)
			items, err := fn(arg, cfg.limit)
			if err != nil {
				return err
			}
			return output(items)
		},
	}
	cmd.Flags().StringVarP(&arg, argName, argShort, "", argName)
	cmd.MarkFlagRequired(argName)
	root.AddCommand(cmd)
}

func output(items []scraper.NewsItem) error {
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "No items scraped.")
		return nil
	}

	var f scraper.OutputFormat
	switch cfg.format {
	case "markdown":
		f = scraper.FormatMarkdown
	case "json":
		f = scraper.FormatJSON
	default:
		f = scraper.FormatText
	}

	result, err := scraper.FormatOutput(items, f)
	if err != nil {
		return err
	}

	if cfg.output != "" {
		if err := os.WriteFile(cfg.output, []byte(result), 0644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Saved %d items to %s\n", len(items), cfg.output)
	} else {
		fmt.Println(result)
	}
	return nil
}
