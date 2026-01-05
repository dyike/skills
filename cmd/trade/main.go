package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dyike/skills/internal/trade"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "market":
		if len(os.Args) < 3 {
			fmt.Println("Usage: trade market <subcommand> [options]")
			fmt.Println("Subcommands: data, indicators")
			os.Exit(1)
		}
		handleMarket(os.Args[2:])

	case "reddit":
		if len(os.Args) < 3 {
			fmt.Println("Usage: trade reddit <subcommand> [options]")
			fmt.Println("Subcommands: subreddit, search, stock, finance")
			os.Exit(1)
		}
		handleReddit(os.Args[2:])

	case "news":
		if len(os.Args) < 3 {
			fmt.Println("Usage: trade news <subcommand> [options]")
			fmt.Println("Subcommands: search, finance, stock")
			os.Exit(1)
		}
		handleNews(os.Args[2:])

	case "tools":
		handleTools()

	case "execute":
		handleExecute(os.Args[2:])

	case "help", "-h", "--help":
		printUsage()

	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Trade Analysis Tool

Usage: trade <command> <subcommand> [options]

Commands:
  market    Market data operations (requires Longport API credentials)
  reddit    Reddit sentiment analysis
  news      News analysis
  tools     Output all tool definitions as JSON (for LLM registration)
  execute   Execute a specific tool with JSON parameters

Market Subcommands:
  data        Get market OHLCV data
              --symbol    Stock symbol (e.g., AAPL.US)
              --count     Number of days (default: 30)

  indicators  Get technical indicators
              --symbol    Stock symbol (e.g., AAPL.US)
              --date      Current date (YYYY-MM-DD)
              --days      Look-back days (default: 30)

Reddit Subcommands:
  subreddit   Get posts from subreddit
              --name      Subreddit name (e.g., wallstreetbets)
              --sort      Sort method: hot, new, top (default: hot)
              --limit     Number of posts (default: 25)

  search      Search Reddit posts
              --query     Search query
              --subreddit Limit to subreddit (optional)
              --sort      Sort: relevance, hot, top, new (default: relevance)
              --time      Time: hour, day, week, month, year (default: week)
              --limit     Number of results (default: 25)

  stock       Get stock mentions
              --symbol    Stock symbol (e.g., AAPL)

  finance     Get popular finance posts
              --limit     Number of posts (default: 50)

News Subcommands:
  search      Search Google News
              --query     Search query
              --language  Language code (default: en)
              --country   Country code (default: US)
              --limit     Number of articles (default: 20)
              --days      Days to look back (default: 7)

  finance     Get finance news
              --limit     Number of articles (default: 20)

  stock       Get stock-specific news
              --symbol    Stock symbol (e.g., AAPL)
              --limit     Number of articles (default: 15)

Environment Variables (for market data):
  LONGPORT_APP_KEY       Longport API app key
  LONGPORT_APP_SECRET    Longport API app secret
  LONGPORT_ACCESS_TOKEN  Longport API access token

Tool Commands:
  tools       Output all tool definitions as JSON
  execute     Execute a tool
              --tool      Tool name (required)
              --params    JSON parameters (required)

Examples:
  trade market data --symbol AAPL.US --count 30
  trade market indicators --symbol AAPL.US --date 2025-01-04 --days 30
  trade reddit subreddit --name wallstreetbets --sort hot --limit 10
  trade reddit stock --symbol AAPL
  trade news search --query "AAPL stock" --days 7
  trade news stock --symbol TSLA
  trade tools
  trade execute --tool get_reddit_posts --params '{"subreddit":"wallstreetbets","limit":5}'`)
}

func handleMarket(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: trade market <subcommand> [options]")
		os.Exit(1)
	}

	subCmd := args[0]
	opts := parseOptions(args[1:])

	cfg := trade.LongportConfig{
		AppKey:      os.Getenv("LONGPORT_APP_KEY"),
		AppSecret:   os.Getenv("LONGPORT_APP_SECRET"),
		AccessToken: os.Getenv("LONGPORT_ACCESS_TOKEN"),
	}

	client, err := trade.NewMarketClient(cfg)
	if err != nil {
		outputError(fmt.Sprintf("Failed to create market client: %v", err))
		os.Exit(1)
	}

	ctx := context.Background()

	switch subCmd {
	case "data":
		symbol := opts["symbol"]
		if symbol == "" {
			outputError("--symbol is required")
			os.Exit(1)
		}
		count := parseIntOpt(opts["count"], 30)

		data, err := client.GetMarketData(ctx, symbol, count)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get market data: %v", err))
			os.Exit(1)
		}
		outputJSON(data)

	case "indicators":
		symbol := opts["symbol"]
		date := opts["date"]
		if symbol == "" || date == "" {
			outputError("--symbol and --date are required")
			os.Exit(1)
		}
		days := parseIntOpt(opts["days"], 30)

		indicators, err := client.GetStockIndicators(ctx, symbol, date, days)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get indicators: %v", err))
			os.Exit(1)
		}
		outputJSON(indicators)

	default:
		fmt.Printf("Unknown market subcommand: %s\n", subCmd)
		os.Exit(1)
	}
}

func handleReddit(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: trade reddit <subcommand> [options]")
		os.Exit(1)
	}

	subCmd := args[0]
	opts := parseOptions(args[1:])

	client := trade.NewRedditClient()

	switch subCmd {
	case "subreddit":
		name := opts["name"]
		if name == "" {
			outputError("--name is required")
			os.Exit(1)
		}
		sort := opts["sort"]
		if sort == "" {
			sort = "hot"
		}
		limit := parseIntOpt(opts["limit"], 25)

		posts, err := client.GetSubredditPosts(name, sort, limit)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get subreddit posts: %v", err))
			os.Exit(1)
		}
		outputJSON(formatRedditOutput(posts))

	case "search":
		query := opts["query"]
		if query == "" {
			outputError("--query is required")
			os.Exit(1)
		}
		subreddit := opts["subreddit"]
		sort := opts["sort"]
		if sort == "" {
			sort = "relevance"
		}
		timePeriod := opts["time"]
		if timePeriod == "" {
			timePeriod = "week"
		}
		limit := parseIntOpt(opts["limit"], 25)

		posts, err := client.SearchReddit(query, subreddit, sort, timePeriod, limit)
		if err != nil {
			outputError(fmt.Sprintf("Failed to search Reddit: %v", err))
			os.Exit(1)
		}
		outputJSON(formatRedditOutput(posts))

	case "stock":
		symbol := opts["symbol"]
		if symbol == "" {
			outputError("--symbol is required")
			os.Exit(1)
		}

		posts, err := client.GetStockMentions(symbol)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get stock mentions: %v", err))
			os.Exit(1)
		}
		outputJSON(formatRedditOutput(posts))

	case "finance":
		limit := parseIntOpt(opts["limit"], 50)

		posts, err := client.GetPopularFinancePosts(limit)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get finance posts: %v", err))
			os.Exit(1)
		}
		outputJSON(formatRedditOutput(posts))

	default:
		fmt.Printf("Unknown reddit subcommand: %s\n", subCmd)
		os.Exit(1)
	}
}

func handleNews(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: trade news <subcommand> [options]")
		os.Exit(1)
	}

	subCmd := args[0]
	opts := parseOptions(args[1:])

	client := trade.NewGoogleNewsClient()

	switch subCmd {
	case "search":
		query := opts["query"]
		if query == "" {
			outputError("--query is required")
			os.Exit(1)
		}
		language := opts["language"]
		if language == "" {
			language = "en"
		}
		country := opts["country"]
		if country == "" {
			country = "US"
		}
		limit := parseIntOpt(opts["limit"], 20)
		days := parseIntOpt(opts["days"], 7)

		articles, err := client.SearchGoogleNews(query, language, country, limit, days)
		if err != nil {
			outputError(fmt.Sprintf("Failed to search news: %v", err))
			os.Exit(1)
		}
		outputJSON(formatNewsOutput(articles))

	case "finance":
		limit := parseIntOpt(opts["limit"], 20)

		articles, err := client.GetFinanceNews(limit)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get finance news: %v", err))
			os.Exit(1)
		}
		outputJSON(formatNewsOutput(articles))

	case "stock":
		symbol := opts["symbol"]
		if symbol == "" {
			outputError("--symbol is required")
			os.Exit(1)
		}
		limit := parseIntOpt(opts["limit"], 15)

		articles, err := client.GetStockNews(symbol, limit)
		if err != nil {
			outputError(fmt.Sprintf("Failed to get stock news: %v", err))
			os.Exit(1)
		}
		outputJSON(formatNewsOutput(articles))

	default:
		fmt.Printf("Unknown news subcommand: %s\n", subCmd)
		os.Exit(1)
	}
}

func parseOptions(args []string) map[string]string {
	opts := make(map[string]string)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				opts[key] = args[i+1]
				i++
			} else {
				opts[key] = "true"
			}
		}
	}

	return opts
}

func parseIntOpt(value string, defaultVal int) int {
	if value == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(value)
	if err != nil {
		return defaultVal
	}
	return val
}

func outputJSON(data interface{}) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
}

func outputError(message string) {
	result := map[string]string{
		"error": message,
	}
	outputJSON(result)
}

func formatRedditOutput(posts []*trade.RedditPost) *trade.RedditOutput {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Reddit Posts (%d results)\n\n", len(posts)))

	for i, post := range posts {
		result.WriteString(fmt.Sprintf("## %d. %s\n", i+1, post.Title))
		result.WriteString(fmt.Sprintf("**r/%s** | **u/%s** | Score: %d | Comments: %d\n",
			post.Subreddit, post.Author, post.Score, post.Comments))
		result.WriteString(fmt.Sprintf("Created: %s | URL: %s\n",
			post.CreatedAt.Format("2006-01-02 15:04"), post.URL))

		if post.Content != "" {
			content := post.Content
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			result.WriteString(fmt.Sprintf("Preview: %s\n", content))
		}
		result.WriteString("\n---\n\n")
	}

	return &trade.RedditOutput{
		Posts:  posts,
		Result: result.String(),
	}
}

func formatNewsOutput(articles []*trade.NewsArticle) *trade.NewsOutput {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# News Articles (%d results)\n\n", len(articles)))

	now := time.Now()
	var breaking, recent, older []*trade.NewsArticle

	for _, article := range articles {
		hoursSince := now.Sub(article.PublishedAt).Hours()
		if hoursSince <= 2 {
			breaking = append(breaking, article)
		} else if hoursSince <= 24 {
			recent = append(recent, article)
		} else {
			older = append(older, article)
		}
	}

	if len(breaking) > 0 {
		result.WriteString("## Breaking News (Last 2 Hours)\n\n")
		for i, article := range breaking {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
			result.WriteString(fmt.Sprintf("**%s** - %s\n", article.Source, article.PublishedAt.Format("15:04")))
			result.WriteString(fmt.Sprintf("URL: %s\n", article.URL))
			if article.Content != "" {
				result.WriteString(fmt.Sprintf("Summary: %s\n", truncate(article.Content, 150)))
			}
			result.WriteString("\n")
		}
	}

	if len(recent) > 0 {
		result.WriteString("## Today's News\n\n")
		for i, article := range recent {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
			result.WriteString(fmt.Sprintf("**%s** - %s\n", article.Source, article.PublishedAt.Format("15:04")))
			result.WriteString(fmt.Sprintf("URL: %s\n", article.URL))
			if article.Content != "" {
				result.WriteString(fmt.Sprintf("Summary: %s\n", truncate(article.Content, 150)))
			}
			result.WriteString("\n")
		}
	}

	if len(older) > 0 {
		result.WriteString("## Recent Coverage\n\n")
		for i, article := range older {
			result.WriteString(fmt.Sprintf("### %d. %s\n", i+1, article.Title))
			result.WriteString(fmt.Sprintf("**%s** - %s\n", article.Source, article.PublishedAt.Format("2006-01-02")))
			result.WriteString(fmt.Sprintf("URL: %s\n", article.URL))
			result.WriteString("\n")
		}
	}

	return &trade.NewsOutput{
		Articles: articles,
		Result:   result.String(),
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// handleTools outputs all tool definitions as JSON for LLM registration
func handleTools() {
	tools := trade.GetAllTools()
	outputJSON(tools)
}

// handleExecute executes a specific tool with JSON parameters
func handleExecute(args []string) {
	opts := parseOptions(args)

	toolName := opts["tool"]
	if toolName == "" {
		outputError("--tool is required")
		os.Exit(1)
	}

	paramsJSON := opts["params"]
	if paramsJSON == "" {
		paramsJSON = "{}"
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
		outputError(fmt.Sprintf("invalid JSON params: %v", err))
		os.Exit(1)
	}

	// Create market client if credentials are available
	var marketClient *trade.MarketClient
	cfg := trade.LongportConfig{
		AppKey:      os.Getenv("LONGPORT_APP_KEY"),
		AppSecret:   os.Getenv("LONGPORT_APP_SECRET"),
		AccessToken: os.Getenv("LONGPORT_ACCESS_TOKEN"),
	}
	if cfg.AppKey != "" && cfg.AppSecret != "" && cfg.AccessToken != "" {
		var err error
		marketClient, err = trade.NewMarketClient(cfg)
		if err != nil {
			// Log but don't fail - market tools will return appropriate errors
			fmt.Fprintf(os.Stderr, "Warning: Failed to create market client: %v\n", err)
		}
	}

	executor := trade.NewToolExecutor(marketClient)
	ctx := context.Background()
	result := executor.Execute(ctx, toolName, params)

	outputJSON(result)
}
