package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dyike/skills/internal/scraper"
	scraper_model "github.com/dyike/skills/models/scraper"
)

type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	sources        stringSlice
	newsletterURL  string
	substackName   string
	tldrCategory   string
	twitterUser    string
	nitterInstance string
	limit          int
	format         string
	useBrowser     bool
	output         string
)

func init() {
	flag.Var(&sources, "source", "Sources to scrape (hn, ph, newsletter, substack, tldr, twitter-trending, twitter-user). Can be specified multiple times.")
	flag.StringVar(&newsletterURL, "newsletter-url", "", "Newsletter archive URL (required for 'newsletter' source)")
	flag.StringVar(&substackName, "substack-name", "", "Substack publication name (required for 'substack' source)")
	flag.StringVar(&tldrCategory, "tldr-category", "tech", "TLDR category: tech, webdev, ai, crypto, devops, founders")
	flag.StringVar(&twitterUser, "twitter-user", "", "Twitter username (required for 'twitter-user' source)")
	flag.StringVar(&nitterInstance, "nitter", "nitter.net", "Nitter instance for Twitter scraping")
	flag.IntVar(&limit, "limit", 20, "Max items per source")
	flag.StringVar(&format, "format", "text", "Output format: text, markdown, json")
	flag.BoolVar(&useBrowser, "use-browser", false, "Use Playwright browser (slower but handles JS)")
	flag.StringVar(&output, "output", "", "Output file path (prints to stdout if not specified)")
	flag.StringVar(&output, "o", "", "Output file path (short)")
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `News Scraper - Fetch latest content from HN, Product Hunt, newsletters, and more

Usage:
  news-scraper [options]

Options:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Examples:
  news-scraper -source hn -limit 20
  news-scraper -source hn -source ph -format markdown
  news-scraper -source newsletter -newsletter-url https://example.substack.com/archive
  news-scraper -source substack -substack-name stratechery
  news-scraper -source tldr -tldr-category ai
  news-scraper -source twitter-trending -nitter nitter.net
  news-scraper -source twitter-user -twitter-user elonmusk
  news-scraper -source hn -source ph -use-browser
`)
	}

	flag.Parse()

	// Default to HN if no sources specified
	if len(sources) == 0 {
		sources = []string{"hn"}
	}

	// Validate sources
	validSources := map[string]bool{
		"hn":               true,
		"ph":               true,
		"newsletter":       true,
		"substack":         true,
		"tldr":             true,
		"twitter-trending": true,
		"twitter-user":     true,
	}

	for _, s := range sources {
		if !validSources[s] {
			fmt.Fprintf(os.Stderr, "Error: invalid source '%s'. Valid sources: hn, ph, newsletter, substack, tldr, twitter-trending, twitter-user\n", s)
			os.Exit(1)
		}
	}

	// Validate TLDR category
	validCategories := map[string]bool{
		"tech":     true,
		"webdev":   true,
		"ai":       true,
		"crypto":   true,
		"devops":   true,
		"founders": true,
	}
	if !validCategories[tldrCategory] {
		fmt.Fprintf(os.Stderr, "Error: invalid TLDR category '%s'\n", tldrCategory)
		os.Exit(1)
	}

	// Validate format
	var outputFormat scraper.OutputFormat
	switch format {
	case "text":
		outputFormat = scraper.FormatText
	case "markdown":
		outputFormat = scraper.FormatMarkdown
	case "json":
		outputFormat = scraper.FormatJSON
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid format '%s'. Valid formats: text, markdown, json\n", format)
		os.Exit(1)
	}

	var allItems []scraper_model.NewsItem

	for _, source := range sources {
		var items []scraper_model.NewsItem
		var err error

		switch source {
		case "hn":
			fmt.Fprintln(os.Stderr, "Scraping Hacker News...")
			if useBrowser {
				items, err = scraper.ScrapeHNPlaywright(limit)
			} else {
				items, err = scraper.ScrapeHNHTTP(limit)
			}

		case "ph":
			fmt.Fprintln(os.Stderr, "Scraping Product Hunt...")
			if useBrowser {
				items, err = scraper.ScrapePHPlaywright(limit)
			} else {
				items, err = scraper.ScrapePHHTTP(limit)
			}

		case "newsletter":
			if newsletterURL == "" {
				fmt.Fprintln(os.Stderr, "Error: -newsletter-url required for 'newsletter' source")
				continue
			}
			fmt.Fprintf(os.Stderr, "Scraping newsletter: %s\n", newsletterURL)
			if useBrowser {
				items, err = scraper.ScrapeNewsletterPlaywright(newsletterURL, nil, limit)
			} else {
				items, err = scraper.ScrapeNewsletterHTTP(newsletterURL, nil, limit)
			}

		case "substack":
			if substackName == "" {
				fmt.Fprintln(os.Stderr, "Error: -substack-name required for 'substack' source")
				continue
			}
			fmt.Fprintf(os.Stderr, "Scraping Substack: %s\n", substackName)
			items, err = scraper.ScrapeSubstackHTTP(substackName, limit)

		case "tldr":
			fmt.Fprintf(os.Stderr, "Scraping TLDR %s...\n", tldrCategory)
			items, err = scraper.ScrapeTLDRHTTP(tldrCategory, limit)

		case "twitter-trending":
			fmt.Fprintf(os.Stderr, "Scraping Twitter trending (via %s)...\n", nitterInstance)
			if useBrowser {
				items, err = scraper.ScrapeTwitterTrendingPlaywright(nitterInstance, limit)
			} else {
				items, err = scraper.ScrapeTwitterTrendingHTTP(nitterInstance, limit)
			}

		case "twitter-user":
			if twitterUser == "" {
				fmt.Fprintln(os.Stderr, "Error: -twitter-user required for 'twitter-user' source")
				continue
			}
			fmt.Fprintf(os.Stderr, "Scraping Twitter user @%s (via %s)...\n", twitterUser, nitterInstance)
			if useBrowser {
				items, err = scraper.ScrapeTwitterUserPlaywright(nitterInstance, twitterUser, limit)
			} else {
				items, err = scraper.ScrapeTwitterUserHTTP(nitterInstance, twitterUser, limit)
			}
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scraping %s: %v\n", source, err)
			continue
		}

		allItems = append(allItems, items...)
	}

	if len(allItems) == 0 {
		fmt.Fprintln(os.Stderr, "No items scraped.")
		os.Exit(1)
	}

	result, err := scraper.FormatOutput(allItems, outputFormat)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	if output != "" {
		err := os.WriteFile(output, []byte(result), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error writing to file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Saved %d items to %s\n", len(allItems), output)
	} else {
		fmt.Println(result)
	}
}
