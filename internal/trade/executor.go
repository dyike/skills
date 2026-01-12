package trade

import (
	"context"
	"fmt"
)

// ToolCallRequest represents an LLM tool call request
type ToolCallRequest struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolCallResponse represents the result of a tool call
type ToolCallResponse struct {
	Success bool        `json:"success"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ToolExecutor handles execution of trade tools
type ToolExecutor struct {
	marketClient *MarketClient
	redditClient *RedditClient
	newsClient   *GoogleNewsClient
}

// NewToolExecutor creates a new tool executor with optional market client
// marketClient can be nil if Longport credentials are not available
func NewToolExecutor(marketClient *MarketClient) *ToolExecutor {
	return &ToolExecutor{
		marketClient: marketClient,
		redditClient: NewRedditClient(),
		newsClient:   NewGoogleNewsClient(),
	}
}

// Execute runs a tool by name with the given parameters
func (te *ToolExecutor) Execute(ctx context.Context, toolName string, params map[string]interface{}) *ToolCallResponse {
	switch toolName {
	case "get_market_data":
		return te.executeGetMarketData(ctx, params)
	case "get_stock_indicators":
		return te.executeGetStockIndicators(ctx, params)
	case "get_reddit_posts":
		return te.executeGetRedditPosts(params)
	case "search_reddit":
		return te.executeSearchReddit(params)
	case "get_stock_mentions":
		return te.executeGetStockMentions(params)
	case "get_finance_posts":
		return te.executeGetFinancePosts(params)
	case "search_news":
		return te.executeSearchNews(params)
	case "get_stock_news":
		return te.executeGetStockNews(params)
	case "get_finance_news":
		return te.executeGetFinanceNews(params)
	default:
		return &ToolCallResponse{
			Success: false,
			Error:   fmt.Sprintf("unknown tool: %s", toolName),
		}
	}
}

// Helper functions for parameter extraction
func getString(params map[string]interface{}, key, defaultVal string) string {
	if val, ok := params[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultVal
}

func getInt(params map[string]interface{}, key string, defaultVal int) int {
	if val, ok := params[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return defaultVal
}

// Market Data Tools

func (te *ToolExecutor) executeGetMarketData(ctx context.Context, params map[string]interface{}) *ToolCallResponse {
	if te.marketClient == nil {
		return &ToolCallResponse{
			Success: false,
			Error:   "market client not initialized - Longport API credentials required",
		}
	}

	symbol := getString(params, "symbol", "")
	if symbol == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "symbol is required",
		}
	}

	count := getInt(params, "count", 30)

	data, err := te.marketClient.GetMarketData(ctx, symbol, count)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  data,
	}
}

func (te *ToolExecutor) executeGetStockIndicators(ctx context.Context, params map[string]interface{}) *ToolCallResponse {
	if te.marketClient == nil {
		return &ToolCallResponse{
			Success: false,
			Error:   "market client not initialized - Longport API credentials required",
		}
	}

	symbol := getString(params, "symbol", "")
	date := getString(params, "date", "")

	if symbol == "" || date == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "symbol and date are required",
		}
	}

	days := getInt(params, "days", 30)

	indicators, err := te.marketClient.GetStockIndicators(ctx, symbol, date, days)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  indicators,
	}
}

// Reddit Tools

func (te *ToolExecutor) executeGetRedditPosts(params map[string]interface{}) *ToolCallResponse {
	subreddit := getString(params, "subreddit", "")
	if subreddit == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "subreddit is required",
		}
	}

	sort := getString(params, "sort", "hot")
	limit := getInt(params, "limit", 25)

	posts, err := te.redditClient.GetSubredditPosts(subreddit, sort, limit)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatRedditResult(posts),
	}
}

func (te *ToolExecutor) executeSearchReddit(params map[string]interface{}) *ToolCallResponse {
	query := getString(params, "query", "")
	if query == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "query is required",
		}
	}

	subreddit := getString(params, "subreddit", "")
	sort := getString(params, "sort", "relevance")
	timePeriod := getString(params, "time", "week")
	limit := getInt(params, "limit", 25)

	posts, err := te.redditClient.SearchReddit(query, subreddit, sort, timePeriod, limit)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatRedditResult(posts),
	}
}

func (te *ToolExecutor) executeGetStockMentions(params map[string]interface{}) *ToolCallResponse {
	symbol := getString(params, "symbol", "")
	if symbol == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "symbol is required",
		}
	}

	posts, err := te.redditClient.GetStockMentions(symbol)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatRedditResult(posts),
	}
}

func (te *ToolExecutor) executeGetFinancePosts(params map[string]interface{}) *ToolCallResponse {
	limit := getInt(params, "limit", 50)

	posts, err := te.redditClient.GetPopularFinancePosts(limit)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatRedditResult(posts),
	}
}

// News Tools

func (te *ToolExecutor) executeSearchNews(params map[string]interface{}) *ToolCallResponse {
	query := getString(params, "query", "")
	if query == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "query is required",
		}
	}

	language := getString(params, "language", "en")
	country := getString(params, "country", "US")
	limit := getInt(params, "limit", 20)
	days := getInt(params, "days", 7)

	articles, err := te.newsClient.SearchGoogleNews(query, language, country, limit, days)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatNewsResult(articles),
	}
}

func (te *ToolExecutor) executeGetStockNews(params map[string]interface{}) *ToolCallResponse {
	symbol := getString(params, "symbol", "")
	if symbol == "" {
		return &ToolCallResponse{
			Success: false,
			Error:   "symbol is required",
		}
	}

	limit := getInt(params, "limit", 15)

	articles, err := te.newsClient.GetStockNews(symbol, limit)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatNewsResult(articles),
	}
}

func (te *ToolExecutor) executeGetFinanceNews(params map[string]interface{}) *ToolCallResponse {
	limit := getInt(params, "limit", 20)

	articles, err := te.newsClient.GetFinanceNews(limit)
	if err != nil {
		return &ToolCallResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	return &ToolCallResponse{
		Success: true,
		Result:  formatNewsResult(articles),
	}
}

// Result formatters

func formatRedditResult(posts []*RedditPost) *RedditOutput {
	var result string
	for i, post := range posts {
		result += fmt.Sprintf("%d. [%s] %s (Score: %d, Comments: %d)\n",
			i+1, post.Subreddit, post.Title, post.Score, post.Comments)
		if post.Content != "" && len(post.Content) > 200 {
			result += fmt.Sprintf("   Preview: %s...\n", post.Content[:200])
		} else if post.Content != "" {
			result += fmt.Sprintf("   Preview: %s\n", post.Content)
		}
		result += fmt.Sprintf("   URL: %s\n\n", post.URL)
	}

	return &RedditOutput{
		Posts:  posts,
		Result: result,
	}
}

func formatNewsResult(articles []*NewsArticle) *NewsOutput {
	var result string
	for i, article := range articles {
		result += fmt.Sprintf("%d. %s\n", i+1, article.Title)
		if article.PublishedAt.IsZero() {
			result += fmt.Sprintf("   Source: %s | Published: Unknown\n", article.Source)
		} else {
			result += fmt.Sprintf("   Source: %s | Published: %s\n",
				article.Source, article.PublishedAt.Format("2006-01-02 15:04"))
		}
		if article.Content != "" && len(article.Content) > 150 {
			result += fmt.Sprintf("   Summary: %s...\n", article.Content[:150])
		} else if article.Content != "" {
			result += fmt.Sprintf("   Summary: %s\n", article.Content)
		}
		result += fmt.Sprintf("   URL: %s\n\n", article.URL)
	}

	return &NewsOutput{
		Articles: articles,
		Result:   result,
	}
}
