package trade

// Tool represents an LLM function tool definition
// Compatible with OpenAI, Anthropic, and Google Gemini function calling formats
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  ToolParameters `json:"parameters"`
}

// ToolParameters defines the parameter schema for a tool
type ToolParameters struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required"`
}

// ToolProperty defines a single parameter property
type ToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
	Default     any      `json:"default,omitempty"`
}

// GetAllTools returns all available trade tools for LLM registration
func GetAllTools() []Tool {
	return []Tool{
		getMarketDataTool(),
		getStockIndicatorsTool(),
		getRedditPostsTool(),
		searchRedditTool(),
		getStockMentionsTool(),
		getFinancePostsTool(),
		searchNewsTool(),
		getStockNewsTool(),
		getFinanceNewsTool(),
	}
}

// getMarketDataTool returns the tool definition for fetching market OHLCV data
func getMarketDataTool() Tool {
	return Tool{
		Name:        "get_market_data",
		Description: "Get OHLCV (Open, High, Low, Close, Volume) candlestick data for a stock symbol. Requires Longport API credentials.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"symbol": {
					Type:        "string",
					Description: "Stock symbol with market suffix (e.g., 'AAPL.US' for US stocks, '700.HK' for Hong Kong stocks)",
				},
				"count": {
					Type:        "integer",
					Description: "Number of trading days to retrieve (default: 30, max: 1000)",
					Default:     30,
				},
			},
			Required: []string{"symbol"},
		},
	}
}

// getStockIndicatorsTool returns the tool definition for technical indicators
func getStockIndicatorsTool() Tool {
	return Tool{
		Name:        "get_stock_indicators",
		Description: "Calculate technical indicators (EMA, SMA, RSI, MACD, Bollinger Bands, ATR) for a stock. Requires Longport API credentials.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"symbol": {
					Type:        "string",
					Description: "Stock symbol with market suffix (e.g., 'AAPL.US', 'TSLA.US')",
				},
				"date": {
					Type:        "string",
					Description: "Current date for analysis in YYYY-MM-DD format (e.g., '2025-01-04')",
				},
				"days": {
					Type:        "integer",
					Description: "Number of look-back days for indicator calculation (default: 30)",
					Default:     30,
				},
			},
			Required: []string{"symbol", "date"},
		},
	}
}

// getRedditPostsTool returns the tool definition for fetching subreddit posts
func getRedditPostsTool() Tool {
	return Tool{
		Name:        "get_reddit_posts",
		Description: "Get posts from a specific subreddit. Useful for sentiment analysis on finance-related subreddits like wallstreetbets, stocks, investing.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"subreddit": {
					Type:        "string",
					Description: "Subreddit name without 'r/' prefix (e.g., 'wallstreetbets', 'stocks', 'investing')",
				},
				"sort": {
					Type:        "string",
					Description: "Sort method for posts",
					Enum:        []string{"hot", "new", "top", "rising"},
					Default:     "hot",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of posts to retrieve (default: 25, max: 100)",
					Default:     25,
				},
			},
			Required: []string{"subreddit"},
		},
	}
}

// searchRedditTool returns the tool definition for searching Reddit
func searchRedditTool() Tool {
	return Tool{
		Name:        "search_reddit",
		Description: "Search Reddit posts by keyword query. Can optionally filter by subreddit and time period.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"query": {
					Type:        "string",
					Description: "Search query (e.g., 'AAPL earnings', '$TSLA', 'stock market crash')",
				},
				"subreddit": {
					Type:        "string",
					Description: "Optional: limit search to specific subreddit(s), comma-separated (e.g., 'wallstreetbets+stocks')",
				},
				"sort": {
					Type:        "string",
					Description: "Sort method for results",
					Enum:        []string{"relevance", "hot", "top", "new", "comments"},
					Default:     "relevance",
				},
				"time": {
					Type:        "string",
					Description: "Time period filter",
					Enum:        []string{"hour", "day", "week", "month", "year", "all"},
					Default:     "week",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of results (default: 25, max: 100)",
					Default:     25,
				},
			},
			Required: []string{"query"},
		},
	}
}

// getStockMentionsTool returns the tool definition for finding stock mentions on Reddit
func getStockMentionsTool() Tool {
	return Tool{
		Name:        "get_stock_mentions",
		Description: "Find Reddit posts mentioning a specific stock symbol across popular finance subreddits (wallstreetbets, stocks, investing, etc.).",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"symbol": {
					Type:        "string",
					Description: "Stock ticker symbol without market suffix (e.g., 'AAPL', 'TSLA', 'GME')",
				},
			},
			Required: []string{"symbol"},
		},
	}
}

// getFinancePostsTool returns the tool definition for popular finance posts
func getFinancePostsTool() Tool {
	return Tool{
		Name:        "get_finance_posts",
		Description: "Get popular posts from major finance subreddits (wallstreetbets, investing, stocks, SecurityAnalysis, ValueInvesting, options, StockMarket).",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"limit": {
					Type:        "integer",
					Description: "Maximum number of posts to retrieve (default: 50)",
					Default:     50,
				},
			},
			Required: []string{},
		},
	}
}

// searchNewsTool returns the tool definition for searching Google News
func searchNewsTool() Tool {
	return Tool{
		Name:        "search_news",
		Description: "Search Google News for articles matching a query. Supports language and country filtering.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"query": {
					Type:        "string",
					Description: "Search query (e.g., 'Apple stock', 'Federal Reserve rate decision', 'tech earnings')",
				},
				"language": {
					Type:        "string",
					Description: "Language code (default: 'en')",
					Default:     "en",
				},
				"country": {
					Type:        "string",
					Description: "Country code for localized results (default: 'US')",
					Default:     "US",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of articles (default: 20)",
					Default:     20,
				},
				"days": {
					Type:        "integer",
					Description: "Number of days to look back (default: 7)",
					Default:     7,
				},
			},
			Required: []string{"query"},
		},
	}
}

// getStockNewsTool returns the tool definition for stock-specific news
func getStockNewsTool() Tool {
	return Tool{
		Name:        "get_stock_news",
		Description: "Get recent news articles about a specific stock symbol. Searches for company news, earnings reports, and analyst coverage.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"symbol": {
					Type:        "string",
					Description: "Stock ticker symbol (e.g., 'AAPL', 'TSLA', 'MSFT')",
				},
				"limit": {
					Type:        "integer",
					Description: "Maximum number of articles (default: 15)",
					Default:     15,
				},
			},
			Required: []string{"symbol"},
		},
	}
}

// getFinanceNewsTool returns the tool definition for general finance news
func getFinanceNewsTool() Tool {
	return Tool{
		Name:        "get_finance_news",
		Description: "Get general finance and market news covering stock market, trading, and financial trends.",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolProperty{
				"limit": {
					Type:        "integer",
					Description: "Maximum number of articles (default: 20)",
					Default:     20,
				},
			},
			Required: []string{},
		},
	}
}
