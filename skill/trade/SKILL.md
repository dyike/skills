---
name: trade
description: Multi-agent trading analysis system. Analyzes stocks through market technicals, social sentiment, news, and fundamentals. Features bull/bear debate and risk management review to generate trading decisions.
---

# Trade Analyzer

Multi-agent trading analysis workflow.

## Usage

When user requests trading analysis for a stock:

```
/trade AAPL.US
/trade TSLA.US 2025-01-04
```

## Workflow

Execute agents in order, passing state between them:

### Phase 1: Information Gathering

1. **MarketAnalyst** - Load `references/market_analyst.md`, analyze technical indicators
2. **SocialAnalyst** - Load `references/social_analyst.md`, analyze social sentiment
3. **NewsAnalyst** - Load `references/news_analyst.md`, analyze recent news
4. **FundamentalsAnalyst** - Load `references/fundamentals_analyst.md`, analyze company financials

### Phase 2: Investment Debate (2 rounds)

5. **BullResearcher** - Load `references/bull_researcher.md`, make bullish case
6. **BearResearcher** - Load `references/bear_researcher.md`, make bearish case
7. **ResearchManager** - Load `references/research_manager.md`, synthesize and recommend

### Phase 3: Trade Execution

8. **Trader** - Load `references/trader.md`, generate specific trade proposal

### Phase 4: Risk Management (3 rounds)

9. **RiskyAnalyst** - Load `references/risky_analyst.md`, aggressive perspective
10. **SafeAnalyst** - Load `references/safe_analyst.md`, conservative perspective
11. **NeutralAnalyst** - Load `references/neutral_analyst.md`, balanced perspective
12. **RiskManager** - Load `references/risk_manager.md`, final BUY/SELL/HOLD decision

## State

Pass these variables between agents:

```
symbol: AAPL.US
trade_date: 2025-01-04
market_report: [from MarketAnalyst]
social_report: [from SocialAnalyst]
news_report: [from NewsAnalyst]
fundamentals_report: [from FundamentalsAnalyst]
debate_history: [accumulated from Bull/Bear]
investment_plan: [from ResearchManager]
trader_plan: [from Trader]
risk_history: [accumulated from Risk analysts]
final_decision: [from RiskManager]
```

## Tools

Agents can call tools via `trade execute --tool <name> --params '<json>'`. Use `trade tools` to get all tool definitions.

### Market Data Tools (requires LONGPORT_* env vars)

| Tool | Parameters | Description |
|------|------------|-------------|
| `get_market_data` | `symbol`, `count` | OHLCV candlestick data |
| `get_stock_indicators` | `symbol`, `date`, `days` | Technical indicators (EMA, SMA, RSI, MACD, Bollinger, ATR) |

**Example - MarketAnalyst:**
```bash
trade execute --tool get_stock_indicators --params '{"symbol":"AAPL.US","date":"2025-01-04","days":30}'
```

### Reddit Tools

| Tool | Parameters | Description |
|------|------------|-------------|
| `get_reddit_posts` | `subreddit`, `sort`, `limit` | Posts from subreddit |
| `search_reddit` | `query`, `subreddit`, `sort`, `time`, `limit` | Search Reddit posts |
| `get_stock_mentions` | `symbol` | Find stock mentions across finance subreddits |
| `get_finance_posts` | `limit` | Popular posts from finance subreddits |

**Example - SocialAnalyst:**
```bash
trade execute --tool get_stock_mentions --params '{"symbol":"AAPL"}'
trade execute --tool get_reddit_posts --params '{"subreddit":"wallstreetbets","sort":"hot","limit":20}'
```

### News Tools

| Tool | Parameters | Description |
|------|------------|-------------|
| `search_news` | `query`, `language`, `country`, `limit`, `days` | Search Google News |
| `get_stock_news` | `symbol`, `limit` | Stock-specific news |
| `get_finance_news` | `limit` | General finance news |

**Example - NewsAnalyst:**
```bash
trade execute --tool get_stock_news --params '{"symbol":"AAPL","limit":15}'
```

## Agent Prompts

All prompts in `references/`:

| Agent | Prompt File |
|-------|-------------|
| MarketAnalyst | `market_analyst.md` |
| SocialAnalyst | `social_analyst.md` |
| NewsAnalyst | `news_analyst.md` |
| FundamentalsAnalyst | `fundamentals_analyst.md` |
| BullResearcher | `bull_researcher.md` |
| BearResearcher | `bear_researcher.md` |
| ResearchManager | `research_manager.md` |
| Trader | `trader.md` |
| RiskyAnalyst | `risky_analyst.md` |
| SafeAnalyst | `safe_analyst.md` |
| NeutralAnalyst | `neutral_analyst.md` |
| RiskManager | `risk_manager.md` |

## Output Format

```markdown
# Trading Analysis: {symbol}

**Date**: {trade_date}

## Market Analysis
{market_report}

## Social Sentiment
{social_report}

## News Analysis
{news_report}

## Fundamentals
{fundamentals_report}

## Investment Debate
{debate_history}

## Investment Plan
{investment_plan}

## Trade Proposal
{trader_plan}

## Risk Assessment
{risk_history}

## Final Decision
{final_decision}
```
