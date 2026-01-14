package scraper

import (
	"fmt"
	"os"

	"github.com/crosszan/modu/repos/scraper"
	"github.com/spf13/cobra"
)

var (
	limit  int
	format string
	output string
)

// NewCmd creates the root scraper command
func NewCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "news-scraper",
		Short: "Fetch latest content from HN, Product Hunt, newsletters, and more",
		Long:  `News Scraper - A tool to fetch and aggregate news from various sources using modu scraper.`,
	}

	// Global flags
	rootCmd.PersistentFlags().IntVarP(&limit, "limit", "l", 20, "Max items to scrape")
	rootCmd.PersistentFlags().StringVarP(&format, "format", "f", "text", "Output format: text, markdown, json")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Output file path (prints to stdout if not specified)")

	// Add subcommands
	rootCmd.AddCommand(
		newHNCmd(),
		newPHCmd(),
		newNewsletterCmd(),
		newTwitterTrendingCmd(),
		newTwitterUserCmd(),
		newXHSCmd(),
	)

	return rootCmd
}

func newHNCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hn",
		Short: "Scrape Hacker News",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "Scraping Hacker News...")
			items, err := scraper.ScrapeHN(limit)
			if err != nil {
				return err
			}
			return outputItems(items)
		},
	}
}

func newPHCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ph",
		Short: "Scrape Product Hunt",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "Scraping Product Hunt...")
			items, err := scraper.ScrapePH(limit)
			if err != nil {
				return err
			}
			return outputItems(items)
		},
	}
}

func newNewsletterCmd() *cobra.Command {
	var url string
	cmd := &cobra.Command{
		Use:   "newsletter",
		Short: "Scrape a newsletter archive",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Scraping newsletter: %s\n", url)
			items, err := scraper.ScrapeNewsletter(url, nil, limit)
			if err != nil {
				return err
			}
			return outputItems(items)
		},
	}
	cmd.Flags().StringVarP(&url, "url", "u", "", "Newsletter archive URL")
	cmd.MarkFlagRequired("url")
	return cmd
}

func newTwitterTrendingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "twitter-trending",
		Short: "Scrape Twitter/X trending topics",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "Scraping Twitter trending from x.com...")
			items, err := scraper.ScrapeTwitterTrending(limit)
			if err != nil {
				return err
			}
			return outputItems(items)
		},
	}
}

func newTwitterUserCmd() *cobra.Command {
	var user string
	cmd := &cobra.Command{
		Use:   "twitter-user",
		Short: "Scrape a Twitter/X user's timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "Scraping Twitter user @%s from x.com...\n", user)
			items, err := scraper.ScrapeTwitterUser(user, limit)
			if err != nil {
				return err
			}
			return outputItems(items)
		},
	}
	cmd.Flags().StringVarP(&user, "user", "u", "", "Twitter username")
	cmd.MarkFlagRequired("user")
	return cmd
}

func newXHSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "xhs",
		Short: "Scrape Xiaohongshu (Little Red Book)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(os.Stderr, "Scraping Xiaohongshu...")
			items, err := scraper.ScrapeXHS(limit)
			if err != nil {
				return err
			}
			return outputItems(items)
		},
	}
}

func outputItems(items []scraper.NewsItem) error {
	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "No items scraped.")
		return nil
	}

	var outputFormat scraper.OutputFormat
	switch format {
	case "markdown":
		outputFormat = scraper.FormatMarkdown
	case "json":
		outputFormat = scraper.FormatJSON
	default:
		outputFormat = scraper.FormatText
	}

	result, err := scraper.FormatOutput(items, outputFormat)
	if err != nil {
		return err
	}

	if output != "" {
		if err := os.WriteFile(output, []byte(result), 0644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Saved %d items to %s\n", len(items), output)
	} else {
		fmt.Println(result)
	}
	return nil
}
