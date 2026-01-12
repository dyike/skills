package trade

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	lpconfig "github.com/longportapp/openapi-go/config"
	"github.com/longportapp/openapi-go/quote"
)

// LongportConfig holds Longport API credentials
type LongportConfig struct {
	AppKey      string
	AppSecret   string
	AccessToken string
}

// MarketClient handles market data operations
type MarketClient struct {
	quoteCtx *quote.QuoteContext
}

// NewMarketClient creates a new market data client
func NewMarketClient(cfg LongportConfig) (*MarketClient, error) {
	if cfg.AppKey == "" || cfg.AppSecret == "" || cfg.AccessToken == "" {
		return nil, errors.New("longport API credentials not configured")
	}

	conf, err := lpconfig.New(lpconfig.WithConfigKey(cfg.AppKey, cfg.AppSecret, cfg.AccessToken))
	if err != nil {
		return nil, err
	}

	quoteContext, err := quote.NewFromCfg(conf)
	if err != nil {
		return nil, err
	}

	return &MarketClient{
		quoteCtx: quoteContext,
	}, nil
}

// GetMarketData retrieves market data for a specific symbol
func (mc *MarketClient) GetMarketData(ctx context.Context, symbol string, count int) ([]*MarketData, error) {
	if mc.quoteCtx == nil {
		return nil, errors.New("quote context is nil")
	}

	if count <= 0 {
		count = 30
	}
	if count > 1000 {
		count = 1000
	}

	sticks, err := mc.quoteCtx.Candlesticks(ctx, symbol, quote.PeriodDay, int32(count), quote.AdjustTypeNo)
	if err != nil {
		return nil, fmt.Errorf("failed to get candlesticks: %w", err)
	}

	return convertCandlesticks(symbol, sticks), nil
}

// GetStockIndicators calculates technical indicators for a stock
func (mc *MarketClient) GetStockIndicators(ctx context.Context, symbol, currDate string, lookBackDays int) (*StockIndicators, error) {
	if mc.quoteCtx == nil {
		return nil, errors.New("quote context is nil")
	}

	// Parse current date
	endDate, err := time.Parse("2006-01-02", currDate)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %s", currDate)
	}

	if lookBackDays <= 0 {
		lookBackDays = 30
	}
	startDate := endDate.AddDate(0, 0, -lookBackDays)

	// Get market data with buffer for indicator calculation
	bufferDays := 250
	bufferStartDate := startDate.AddDate(0, 0, -bufferDays)

	sticks, err := mc.quoteCtx.HistoryCandlesticksByDate(ctx, symbol, quote.PeriodDay, quote.AdjustTypeNo, &bufferStartDate, &endDate)
	if err != nil {
		return nil, err
	}

	marketData := convertCandlesticks(symbol, sticks)
	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data available for symbol %s", symbol)
	}

	// Calculate all indicators
	allIndicators := calculateAllIndicators(marketData, startDate, endDate)

	// Generate summary
	summary := generateTechnicalSummary(allIndicators)

	return &StockIndicators{
		Symbol:     symbol,
		StartDate:  startDate.Format("2006-01-02"),
		EndDate:    currDate,
		Indicators: allIndicators,
		Summary:    summary,
	}, nil
}

func convertCandlesticks(symbol string, sticks []*quote.Candlestick) []*MarketData {
	marketData := make([]*MarketData, 0, len(sticks))
	for _, stick := range sticks {
		date := time.Unix(stick.Timestamp, 0).Format("2006-01-02")
		open, _ := stick.Open.Float64()
		high, _ := stick.High.Float64()
		low, _ := stick.Low.Float64()
		close, _ := stick.Close.Float64()

		marketData = append(marketData, &MarketData{
			Symbol: symbol,
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: stick.Volume,
		})
	}

	return marketData
}

// calculateAllIndicators calculates all technical indicators
func calculateAllIndicators(data []*MarketData, startDate, endDate time.Time) map[string][]IndicatorValue {
	indicators := make(map[string][]IndicatorValue)

	// Sort data by date
	sort.Slice(data, func(i, j int) bool {
		return data[i].Date < data[j].Date
	})

	closes := make([]float64, len(data))
	highs := make([]float64, len(data))
	lows := make([]float64, len(data))
	volumes := make([]float64, len(data))
	dates := make([]string, len(data))

	for i, d := range data {
		closes[i] = d.Close
		highs[i] = d.High
		lows[i] = d.Low
		volumes[i] = float64(d.Volume)
		dates[i] = d.Date
	}

	// Calculate indicators
	indicators["close_10_ema"] = calculateEMA(closes, dates, 10, startDate, endDate)
	indicators["close_50_sma"] = calculateSMA(closes, dates, 50, startDate, endDate)
	indicators["close_200_sma"] = calculateSMA(closes, dates, 200, startDate, endDate)
	indicators["rsi"] = calculateRSI(closes, dates, 14, startDate, endDate)
	indicators["macd"], indicators["macds"], indicators["macdh"] = calculateMACD(closes, dates, startDate, endDate)
	indicators["boll"], indicators["boll_ub"], indicators["boll_lb"] = calculateBollingerBands(closes, dates, 20, 2, startDate, endDate)
	indicators["atr"] = calculateATR(highs, lows, closes, dates, 14, startDate, endDate)

	return indicators
}

func calculateSMA(values []float64, dates []string, period int, startDate, endDate time.Time) []IndicatorValue {
	var result []IndicatorValue

	for i := period - 1; i < len(values); i++ {
		date, _ := time.Parse("2006-01-02", dates[i])
		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += values[j]
		}
		sma := sum / float64(period)

		result = append(result, IndicatorValue{
			Date:  dates[i],
			Value: sma,
		})
	}

	return result
}

func calculateEMA(values []float64, dates []string, period int, startDate, endDate time.Time) []IndicatorValue {
	var result []IndicatorValue

	if len(values) < period {
		return result
	}

	multiplier := 2.0 / float64(period+1)

	// Calculate initial SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += values[i]
	}
	ema := sum / float64(period)

	for i := period; i < len(values); i++ {
		ema = (values[i]-ema)*multiplier + ema

		date, _ := time.Parse("2006-01-02", dates[i])
		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		result = append(result, IndicatorValue{
			Date:  dates[i],
			Value: ema,
		})
	}

	return result
}

func calculateRSI(values []float64, dates []string, period int, startDate, endDate time.Time) []IndicatorValue {
	var result []IndicatorValue

	if len(values) < period+1 {
		return result
	}

	gains := make([]float64, len(values)-1)
	losses := make([]float64, len(values)-1)

	for i := 1; i < len(values); i++ {
		change := values[i] - values[i-1]
		if change > 0 {
			gains[i-1] = change
		} else {
			losses[i-1] = -change
		}
	}

	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := period; i < len(values)-1; i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)

		rs := 0.0
		if avgLoss > 0 {
			rs = avgGain / avgLoss
		}
		rsi := 100 - (100 / (1 + rs))

		date, _ := time.Parse("2006-01-02", dates[i+1])
		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		result = append(result, IndicatorValue{
			Date:  dates[i+1],
			Value: rsi,
		})
	}

	return result
}

func calculateMACD(values []float64, dates []string, startDate, endDate time.Time) (macd, signal, histogram []IndicatorValue) {
	if len(values) < 26 {
		return
	}

	// Calculate EMA12 and EMA26
	ema12 := make([]float64, len(values))
	ema26 := make([]float64, len(values))

	mult12 := 2.0 / 13.0
	mult26 := 2.0 / 27.0

	sum12 := 0.0
	for i := 0; i < 12; i++ {
		sum12 += values[i]
	}
	ema12[11] = sum12 / 12

	for i := 12; i < len(values); i++ {
		ema12[i] = (values[i]-ema12[i-1])*mult12 + ema12[i-1]
	}

	sum26 := 0.0
	for i := 0; i < 26; i++ {
		sum26 += values[i]
	}
	ema26[25] = sum26 / 26

	for i := 26; i < len(values); i++ {
		ema26[i] = (values[i]-ema26[i-1])*mult26 + ema26[i-1]
	}

	// Calculate MACD line
	macdLine := make([]float64, len(values))
	for i := 25; i < len(values); i++ {
		macdLine[i] = ema12[i] - ema26[i]
	}

	// Calculate Signal line (9-period EMA of MACD)
	signalLine := make([]float64, len(values))
	mult9 := 2.0 / 10.0

	sum9 := 0.0
	for i := 25; i < 34 && i < len(values); i++ {
		sum9 += macdLine[i]
	}
	if len(values) > 33 {
		signalLine[33] = sum9 / 9
		for i := 34; i < len(values); i++ {
			signalLine[i] = (macdLine[i]-signalLine[i-1])*mult9 + signalLine[i-1]
		}
	}

	// Build results
	for i := 33; i < len(values); i++ {
		date, _ := time.Parse("2006-01-02", dates[i])
		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		macd = append(macd, IndicatorValue{Date: dates[i], Value: macdLine[i]})
		signal = append(signal, IndicatorValue{Date: dates[i], Value: signalLine[i]})
		histogram = append(histogram, IndicatorValue{Date: dates[i], Value: macdLine[i] - signalLine[i]})
	}

	return
}

func calculateBollingerBands(values []float64, dates []string, period int, stdDev float64, startDate, endDate time.Time) (middle, upper, lower []IndicatorValue) {
	for i := period - 1; i < len(values); i++ {
		date, _ := time.Parse("2006-01-02", dates[i])
		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += values[j]
		}
		sma := sum / float64(period)

		variance := 0.0
		for j := i - period + 1; j <= i; j++ {
			variance += (values[j] - sma) * (values[j] - sma)
		}
		std := math.Sqrt(variance / float64(period))

		middle = append(middle, IndicatorValue{Date: dates[i], Value: sma})
		upper = append(upper, IndicatorValue{Date: dates[i], Value: sma + stdDev*std})
		lower = append(lower, IndicatorValue{Date: dates[i], Value: sma - stdDev*std})
	}

	return
}

func calculateATR(highs, lows, closes []float64, dates []string, period int, startDate, endDate time.Time) []IndicatorValue {
	var result []IndicatorValue

	if len(highs) < period+1 {
		return result
	}

	// Calculate True Range
	tr := make([]float64, len(highs))
	for i := 1; i < len(highs); i++ {
		hl := highs[i] - lows[i]
		hc := math.Abs(highs[i] - closes[i-1])
		lc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hc, lc))
	}

	// Calculate initial ATR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	atr := sum / float64(period)

	for i := period + 1; i < len(highs); i++ {
		atr = (atr*float64(period-1) + tr[i]) / float64(period)

		date, _ := time.Parse("2006-01-02", dates[i])
		if date.Before(startDate) || date.After(endDate) {
			continue
		}

		result = append(result, IndicatorValue{
			Date:  dates[i],
			Value: atr,
		})
	}

	return result
}

func generateTechnicalSummary(indicators map[string][]IndicatorValue) string {
	var summary strings.Builder

	getLatestValue := func(indicator string) (float64, bool) {
		if values, exists := indicators[indicator]; exists && len(values) > 0 {
			return values[len(values)-1].Value, true
		}
		return 0, false
	}

	summary.WriteString("**Trend Analysis:**\n")
	if ema10, exists1 := getLatestValue("close_10_ema"); exists1 {
		if sma50, exists2 := getLatestValue("close_50_sma"); exists2 {
			if ema10 > sma50 {
				summary.WriteString("- Short-term trend is BULLISH (10 EMA > 50 SMA)\n")
			} else {
				summary.WriteString("- Short-term trend is BEARISH (10 EMA < 50 SMA)\n")
			}
		}
	}

	if sma50, exists1 := getLatestValue("close_50_sma"); exists1 {
		if sma200, exists2 := getLatestValue("close_200_sma"); exists2 {
			if sma50 > sma200 {
				summary.WriteString("- Long-term trend is BULLISH (50 SMA > 200 SMA)\n")
			} else {
				summary.WriteString("- Long-term trend is BEARISH (50 SMA < 200 SMA)\n")
			}
		}
	}

	summary.WriteString("\n**Momentum Analysis:**\n")
	if rsi, exists := getLatestValue("rsi"); exists {
		if rsi > 70 {
			summary.WriteString(fmt.Sprintf("- RSI (%.2f) indicates OVERBOUGHT conditions\n", rsi))
		} else if rsi < 30 {
			summary.WriteString(fmt.Sprintf("- RSI (%.2f) indicates OVERSOLD conditions\n", rsi))
		} else {
			summary.WriteString(fmt.Sprintf("- RSI (%.2f) is in NEUTRAL range\n", rsi))
		}
	}

	summary.WriteString("\n**MACD Analysis:**\n")
	if macd, exists1 := getLatestValue("macd"); exists1 {
		if macdSignal, exists2 := getLatestValue("macds"); exists2 {
			if macd > macdSignal {
				summary.WriteString("- MACD line above signal line: BULLISH momentum\n")
			} else {
				summary.WriteString("- MACD line below signal line: BEARISH momentum\n")
			}
		}
	}

	if macdHist, exists := getLatestValue("macdh"); exists {
		if macdHist > 0 {
			summary.WriteString("- MACD Histogram positive: Increasing bullish momentum\n")
		} else {
			summary.WriteString("- MACD Histogram negative: Increasing bearish momentum\n")
		}
	}

	return summary.String()
}
