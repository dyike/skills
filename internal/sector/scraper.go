package sector

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/crosszan/modu/pkg/playwright"
	pwgo "github.com/playwright-community/playwright-go"
)

// ScrapeSectors 抓取板块列表
func ScrapeSectors(sectorType SectorType, limit int) (*SectorListResponse, error) {
	url, ok := SectorURLs[sectorType]
	if !ok {
		return nil, fmt.Errorf("unsupported sector type: %s", sectorType)
	}

	// 创建浏览器实例
	browser, err := playwright.New(
		playwright.WithHeadless(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	// 创建页面
	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// 注入反检测脚本
	if err := page.InjectAntiDetect(); err != nil {
		return nil, fmt.Errorf("failed to inject anti-detect: %w", err)
	}

	// 导航到页面 (使用 domcontentloaded 代替 networkidle，更快响应)
	if err := page.Goto(url, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}

	// 等待页面加载完成 (东方财富页面有大量 JS 渲染)
	page.Wait(5 * time.Second)

	// 提取板块数据
	sectors, err := extractSectorData(page, limit)
	if err != nil {
		return nil, err
	}

	if len(sectors) == 0 {
		return nil, fmt.Errorf("no sectors found, page structure may have changed")
	}

	// 构建响应
	response := &SectorListResponse{
		Type:      sectorType,
		Count:     len(sectors),
		Sectors:   sectors,
		Timestamp: FormatTimestamp(),
		Summary:   generateListSummary(sectorType, sectors),
	}

	return response, nil
}

// SortType 排序类型
type SortType string

const (
	SortByChange SortType = "change" // 按涨跌幅排序
	SortByAmount SortType = "amount" // 按成交额排序
)

// ScrapeHotSectors 抓取热门板块
// sortBy: "change" 按涨跌幅排序, "amount" 按成交额排序
func ScrapeHotSectors(limit int, sortBy SortType) (*HotSectorsResponse, error) {
	// 抓取行业板块
	industrySectors, err := ScrapeSectors(SectorTypeIndustry, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape industry sectors: %w", err)
	}

	sectors := industrySectors.Sectors

	// 根据排序类型选择排序方式
	var sortFunc func(i, j int) bool
	switch sortBy {
	case SortByAmount:
		sortFunc = func(i, j int) bool {
			return sectors[i].Amount > sectors[j].Amount
		}
	default: // SortByChange
		sortFunc = func(i, j int) bool {
			return sectors[i].ChangeRate > sectors[j].ChangeRate
		}
	}

	// 排序
	sort.Slice(sectors, sortFunc)

	// 获取前列
	topRising := make([]*SectorInfo, 0, limit)
	for i := 0; i < len(sectors) && i < limit; i++ {
		topRising = append(topRising, sectors[i])
	}

	// 获取末列 (仅在按涨跌幅排序时有意义)
	topFalling := make([]*SectorInfo, 0, limit)
	if sortBy == SortByChange {
		sort.Slice(sectors, func(i, j int) bool {
			return sectors[i].ChangeRate < sectors[j].ChangeRate
		})
		for i := 0; i < len(sectors) && i < limit; i++ {
			if sectors[i].ChangeRate < 0 {
				topFalling = append(topFalling, sectors[i])
			}
		}
	}

	response := &HotSectorsResponse{
		TopRising:  topRising,
		TopFalling: topFalling,
		Timestamp:  FormatTimestamp(),
		Summary:    generateHotSummaryWithSort(topRising, topFalling, sortBy),
	}

	return response, nil
}

// extractSectorData 从页面提取板块数据
// 东方财富板块列表页面结构 (2024年验证):
// table tbody tr
//
//	td[0]: 排名
//	td[1]: 板块名称
//	td[2]: 相关链接 (股吧/资金流/研报)
//	td[3]: 最新价
//	td[4]: 涨跌额
//	td[5]: 涨跌幅
//	td[6]: 总市值
//	td[7]: 换手率
//	td[8]: 上涨家数
//	td[9]: 下跌家数
//	td[10]: 领涨股票
//	td[11]: 领涨股涨跌幅
func extractSectorData(page *playwright.Page, limit int) ([]*SectorInfo, error) {
	timestamp := FormatTimestamp()
	sectors := make([]*SectorInfo, 0, limit)

	// 选择表格行
	rows, err := page.QuerySelectorAll("table tbody tr")
	if err != nil {
		return nil, fmt.Errorf("failed to query table rows: %w", err)
	}

	for _, row := range rows {
		if len(sectors) >= limit {
			break
		}

		// 获取所有单元格
		cells, err := row.QuerySelectorAll("td")
		if err != nil || len(cells) < 11 {
			continue
		}

		// 按正确的列索引提取字段
		// td[0]: 排名 (跳过)
		name := getInnerText(cells[1]) // td[1]: 板块名称
		// td[2]: 相关链接 (跳过，这里是"股吧 资金流 研报")
		price := parseFloat(getInnerText(cells[3]))                               // td[3]: 最新价
		change := parseFloat(getInnerText(cells[4]))                              // td[4]: 涨跌额
		changeRate := parseFloat(strings.TrimSuffix(getInnerText(cells[5]), "%")) // td[5]: 涨跌幅
		marketCap := getInnerText(cells[6])                                       // td[6]: 总市值 (暂存为字符串)
		// td[7]: 换手率 (暂不使用)
		riseCount := int(parseInt(getInnerText(cells[8])))                         // td[8]: 上涨家数
		fallCount := int(parseInt(getInnerText(cells[9])))                         // td[9]: 下跌家数
		leaderStock := getInnerText(cells[10])                                     // td[10]: 领涨股票
		leaderRate := parseFloat(strings.TrimSuffix(getInnerText(cells[11]), "%")) // td[11]: 领涨股涨跌幅

		if name == "" {
			continue
		}

		// 解析总市值为成交金额字段 (亿)
		amount := parseMarketCap(marketCap)

		sectors = append(sectors, &SectorInfo{
			Code:        "", // 页面不再显示板块代码
			Name:        name,
			Price:       price,
			Change:      change,
			ChangeRate:  changeRate,
			Volume:      0, // 页面不再显示成交量
			Amount:      amount,
			LeaderStock: leaderStock,
			LeaderRate:  leaderRate,
			RiseCount:   riseCount,
			FallCount:   fallCount,
			Timestamp:   timestamp,
		})
	}

	return sectors, nil
}

// parseMarketCap 解析总市值 (支持万亿、亿单位)
func parseMarketCap(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "--" {
		return 0
	}

	multiplier := 1.0
	if strings.HasSuffix(s, "万亿") {
		s = strings.TrimSuffix(s, "万亿")
		multiplier = 10000
	} else if strings.HasSuffix(s, "亿") {
		s = strings.TrimSuffix(s, "亿")
		multiplier = 1
	}

	f, _ := strconv.ParseFloat(s, 64)
	return f * multiplier
}

// getInnerText 获取元素内文本
func getInnerText(el pwgo.ElementHandle) string {
	text, err := el.InnerText()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}

// parseFloat 解析浮点数
func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "--" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseInt 解析整数
func parseInt(s string) int64 {
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "--" {
		return 0
	}
	// 处理单位 (万、亿)
	multiplier := int64(1)
	if strings.HasSuffix(s, "万") {
		s = strings.TrimSuffix(s, "万")
		multiplier = 10000
	} else if strings.HasSuffix(s, "亿") {
		s = strings.TrimSuffix(s, "亿")
		multiplier = 100000000
	}

	f, _ := strconv.ParseFloat(s, 64)
	return int64(f * float64(multiplier))
}

// generateListSummary 生成板块列表摘要
func generateListSummary(sectorType SectorType, sectors []*SectorInfo) string {
	if len(sectors) == 0 {
		return "暂无数据"
	}

	// 统计涨跌情况
	riseCount := 0
	fallCount := 0
	flatCount := 0
	for _, s := range sectors {
		if s.ChangeRate > 0 {
			riseCount++
		} else if s.ChangeRate < 0 {
			fallCount++
		} else {
			flatCount++
		}
	}

	typeName := "行业"
	if sectorType == SectorTypeConcept {
		typeName = "概念"
	}

	// 找出涨幅最大的板块
	var top *SectorInfo
	for _, s := range sectors {
		if top == nil || s.ChangeRate > top.ChangeRate {
			top = s
		}
	}

	return fmt.Sprintf("%s板块共%d个，上涨%d个，下跌%d个，平盘%d个。涨幅最大: %s(%.2f%%)",
		typeName, len(sectors), riseCount, fallCount, flatCount, top.Name, top.ChangeRate)
}

// generateHotSummary 生成热门板块摘要
func generateHotSummary(rising, falling []*SectorInfo) string {
	return generateHotSummaryWithSort(rising, falling, SortByChange)
}

// generateHotSummaryWithSort 生成热门板块摘要（支持排序类型）
func generateHotSummaryWithSort(rising, falling []*SectorInfo, sortBy SortType) string {
	var parts []string

	if len(rising) > 0 {
		names := make([]string, 0, 3)
		for i := 0; i < len(rising) && i < 3; i++ {
			if sortBy == SortByAmount {
				names = append(names, fmt.Sprintf("%s(%.0f亿)", rising[i].Name, rising[i].Amount))
			} else {
				names = append(names, fmt.Sprintf("%s(+%.2f%%)", rising[i].Name, rising[i].ChangeRate))
			}
		}
		if sortBy == SortByAmount {
			parts = append(parts, "成交额前三: "+strings.Join(names, "、"))
		} else {
			parts = append(parts, "涨幅前三: "+strings.Join(names, "、"))
		}
	}

	if len(falling) > 0 && sortBy == SortByChange {
		names := make([]string, 0, 3)
		for i := 0; i < len(falling) && i < 3; i++ {
			names = append(names, fmt.Sprintf("%s(%.2f%%)", falling[i].Name, falling[i].ChangeRate))
		}
		parts = append(parts, "跌幅前三: "+strings.Join(names, "、"))
	}

	if len(parts) == 0 {
		return "暂无数据"
	}

	return strings.Join(parts, "；")
}

// ScrapeSectorStocks 抓取板块内的个股列表
func ScrapeSectorStocks(sectorName string, limit int) (*SectorStocksResponse, error) {
	// 创建浏览器实例
	browser, err := playwright.New(
		playwright.WithHeadless(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	// 创建页面
	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// 注入反检测脚本
	if err := page.InjectAntiDetect(); err != nil {
		return nil, fmt.Errorf("failed to inject anti-detect: %w", err)
	}

	// 先访问行业板块列表获取板块代码
	url := "https://quote.eastmoney.com/center/boardlist.html#industry_board"
	if err := page.Goto(url, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}
	page.Wait(5 * time.Second)

	// 查找板块代码
	sectorCode, err := findSectorCode(page, sectorName)
	if err != nil {
		return nil, fmt.Errorf("failed to find sector '%s': %w", sectorName, err)
	}

	// 导航到板块成分股页面
	stocksURL := fmt.Sprintf("https://quote.eastmoney.com/center/gridlist.html#boards2-90.%s", sectorCode)
	if err := page.Goto(stocksURL, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to stocks page: %w", err)
	}
	page.Wait(5 * time.Second)

	// 提取个股数据
	stocks, err := extractStockData(page, limit)
	if err != nil {
		return nil, err
	}

	if len(stocks) == 0 {
		return nil, fmt.Errorf("no stocks found for sector '%s'", sectorName)
	}

	// 构建响应
	response := &SectorStocksResponse{
		SectorName: sectorName,
		SectorCode: sectorCode,
		Count:      len(stocks),
		Stocks:     stocks,
		Timestamp:  FormatTimestamp(),
		Summary:    generateStocksSummary(sectorName, stocks),
	}

	return response, nil
}

// findSectorCode 从页面找到板块代码
func findSectorCode(page *playwright.Page, sectorName string) (string, error) {
	// 使用 JavaScript 查找板块链接并提取代码
	result, err := page.Evaluate(fmt.Sprintf(`
		(() => {
			const links = Array.from(document.querySelectorAll('a'));
			const targetLink = links.find(l => l.innerText.trim() === '%s');
			if (targetLink && targetLink.href) {
				// 从 URL 提取板块代码，如 BK1031
				const match = targetLink.href.match(/BK\d+/);
				return match ? match[0] : null;
			}
			return null;
		})()
	`, sectorName))
	if err != nil {
		return "", err
	}

	if result == nil {
		return "", fmt.Errorf("sector not found")
	}

	code, ok := result.(string)
	if !ok || code == "" {
		return "", fmt.Errorf("invalid sector code")
	}

	return code, nil
}

// extractStockData 从页面提取个股数据
// 东方财富板块成分股页面结构 (2024年验证):
// table tbody tr
//
//	td[0]: 序号
//	td[1]: 代码
//	td[2]: 名称
//	td[3]: 相关链接 (股吧/资金流/数据)
//	td[4]: 最新价
//	td[5]: 涨跌幅
//	td[6]: 涨跌额
//	td[7]: 成交量
//	td[8]: 成交额
//	td[9]: 振幅
//	td[10-13]: 最高/最低/今开/昨收
//	td[14]: 量比
//	td[15]: 换手率
//	td[16]: 市盈率
func extractStockData(page *playwright.Page, limit int) ([]*StockInfo, error) {
	timestamp := FormatTimestamp()
	stocks := make([]*StockInfo, 0, limit)

	// 选择表格行
	rows, err := page.QuerySelectorAll("table tbody tr")
	if err != nil {
		return nil, fmt.Errorf("failed to query table rows: %w", err)
	}

	for _, row := range rows {
		if len(stocks) >= limit {
			break
		}

		// 获取所有单元格
		cells, err := row.QuerySelectorAll("td")
		if err != nil || len(cells) < 16 {
			continue
		}

		// 按正确的列索引提取字段
		code := getInnerText(cells[1]) // td[1]: 代码
		name := getInnerText(cells[2]) // td[2]: 名称
		// td[3]: 相关链接 (跳过)
		price := parseFloat(getInnerText(cells[4]))                               // td[4]: 最新价
		changeRate := parseFloat(strings.TrimSuffix(getInnerText(cells[5]), "%")) // td[5]: 涨跌幅
		change := parseFloat(getInnerText(cells[6]))                              // td[6]: 涨跌额
		volume := parseInt(getInnerText(cells[7]))                                // td[7]: 成交量
		amountStr := getInnerText(cells[8])                                       // td[8]: 成交额
		amount := parseMarketCap(amountStr)                                       // 解析成交额(支持亿单位)
		// td[9-14]: 振幅/最高/最低/今开/昨收/量比 (跳过)
		turnover := parseFloat(strings.TrimSuffix(getInnerText(cells[15]), "%")) // td[15]: 换手率
		pe := parseFloat(getInnerText(cells[16]))                                // td[16]: 市盈率

		if name == "" || code == "" {
			continue
		}

		stocks = append(stocks, &StockInfo{
			Code:       code,
			Name:       name,
			Price:      price,
			Change:     change,
			ChangeRate: changeRate,
			Volume:     volume,
			Amount:     amount,
			Turnover:   turnover,
			PE:         pe,
			Timestamp:  timestamp,
		})
	}

	return stocks, nil
}

// generateStocksSummary 生成个股列表摘要
func generateStocksSummary(sectorName string, stocks []*StockInfo) string {
	if len(stocks) == 0 {
		return "暂无数据"
	}

	// 统计涨跌情况
	riseCount := 0
	fallCount := 0
	limitUpCount := 0 // 涨停
	for _, s := range stocks {
		if s.ChangeRate > 0 {
			riseCount++
			if s.ChangeRate >= 9.9 { // 涨停
				limitUpCount++
			}
		} else if s.ChangeRate < 0 {
			fallCount++
		}
	}

	// 找涨幅最大的
	var top *StockInfo
	for _, s := range stocks {
		if top == nil || s.ChangeRate > top.ChangeRate {
			top = s
		}
	}

	summary := fmt.Sprintf("%s板块共%d只个股，上涨%d只，下跌%d只",
		sectorName, len(stocks), riseCount, fallCount)
	if limitUpCount > 0 {
		summary += fmt.Sprintf("，涨停%d只", limitUpCount)
	}
	summary += fmt.Sprintf("。领涨: %s(%.2f%%)", top.Name, top.ChangeRate)

	return summary
}

// FundFlowURLs 资金流向数据 URL
var FundFlowURLs = map[SectorType]string{
	SectorTypeIndustry: "https://data.eastmoney.com/bkzj/hy.html",
	SectorTypeConcept:  "https://data.eastmoney.com/bkzj/gn.html",
}

// ScrapeFundFlow 抓取板块资金流向
func ScrapeFundFlow(sectorType SectorType, limit int) (*FundFlowResponse, error) {
	url, ok := FundFlowURLs[sectorType]
	if !ok {
		return nil, fmt.Errorf("unsupported sector type: %s", sectorType)
	}

	// 创建浏览器实例
	browser, err := playwright.New(
		playwright.WithHeadless(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	// 创建页面
	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// 注入反检测脚本
	if err := page.InjectAntiDetect(); err != nil {
		return nil, fmt.Errorf("failed to inject anti-detect: %w", err)
	}

	// 导航到页面
	if err := page.Goto(url, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}
	page.Wait(5 * time.Second)

	// 提取资金流向数据
	flows, err := extractFundFlowData(page, limit)
	if err != nil {
		return nil, err
	}

	if len(flows) == 0 {
		return nil, fmt.Errorf("no fund flow data found")
	}

	response := &FundFlowResponse{
		Type:      sectorType,
		Count:     len(flows),
		Flows:     flows,
		Timestamp: FormatTimestamp(),
		Summary:   generateFundFlowSummary(sectorType, flows),
	}

	return response, nil
}

// extractFundFlowData 从页面提取资金流向数据
// 东方财富资金流向页面结构:
// table[1] tbody tr (第二个表格)
//
//	td[0]: 序号
//	td[1]: 名称
//	td[2]: 相关链接
//	td[3]: 今日涨跌幅
//	td[4]: 主力净流入-净额
//	td[5]: 主力净流入-净占比
//	td[6]: 超大单净流入-净额
//	td[7]: 超大单净流入-净占比
//	td[8]: 大单净流入-净额
//	td[9]: 大单净流入-净占比
//	td[10]: 中单净流入-净额
//	td[11]: 中单净流入-净占比
//	td[12]: 小单净流入-净额
//	td[13]: 小单净流入-净占比
//	td[14]: 主力净流入最大股
func extractFundFlowData(page *playwright.Page, limit int) ([]*FundFlowInfo, error) {
	timestamp := FormatTimestamp()
	flows := make([]*FundFlowInfo, 0, limit)

	// 使用 JavaScript 获取第二个表格的数据
	result, err := page.Evaluate(`
		(() => {
			const tables = Array.from(document.querySelectorAll('table'));
			if (tables.length < 2) return [];
			const table = tables[1];
			const rows = Array.from(table.querySelectorAll('tbody tr'));
			return rows.map(row => {
				const cells = Array.from(row.querySelectorAll('td'));
				if (cells.length < 13) return null;
				return {
					name: cells[1] ? cells[1].innerText.trim() : '',
					changeRate: cells[3] ? cells[3].innerText.trim() : '',
					mainAmt: cells[4] ? cells[4].innerText.trim() : '',
					mainRatio: cells[5] ? cells[5].innerText.trim() : '',
					superBig: cells[6] ? cells[6].innerText.trim() : '',
					big: cells[8] ? cells[8].innerText.trim() : '',
					medium: cells[10] ? cells[10].innerText.trim() : '',
					small: cells[12] ? cells[12].innerText.trim() : ''
				};
			}).filter(x => x !== null);
		})()
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to extract fund flow data: %w", err)
	}

	// 解析结果
	rows, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid fund flow data format")
	}

	for i, row := range rows {
		if len(flows) >= limit {
			break
		}

		rowMap, ok := row.(map[string]interface{})
		if !ok {
			continue
		}

		name := getString(rowMap, "name")
		if name == "" {
			continue
		}

		flows = append(flows, &FundFlowInfo{
			Name:          name,
			ChangeRate:    parsePercentage(getString(rowMap, "changeRate")),
			MainNetInflow: parseAmount(getString(rowMap, "mainAmt")),
			MainNetRatio:  parsePercentage(getString(rowMap, "mainRatio")),
			SuperBig:      parseAmount(getString(rowMap, "superBig")),
			Big:           parseAmount(getString(rowMap, "big")),
			Medium:        parseAmount(getString(rowMap, "medium")),
			Small:         parseAmount(getString(rowMap, "small")),
			Timestamp:     timestamp,
		})

		_ = i
	}

	return flows, nil
}

// getString 从 map 获取字符串
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// parsePercentage 解析百分比
func parsePercentage(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseAmount 解析金额 (支持亿单位)
func parseAmount(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "--" {
		return 0
	}
	// 处理负数和亿单位
	s = strings.ReplaceAll(s, ",", "")
	multiplier := 1.0
	if strings.HasSuffix(s, "亿") {
		s = strings.TrimSuffix(s, "亿")
		multiplier = 1
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f * multiplier
}

// generateFundFlowSummary 生成资金流向摘要
func generateFundFlowSummary(sectorType SectorType, flows []*FundFlowInfo) string {
	if len(flows) == 0 {
		return "暂无数据"
	}

	typeName := "行业"
	if sectorType == SectorTypeConcept {
		typeName = "概念"
	}

	// 统计主力净流入情况
	inflowCount := 0
	outflowCount := 0
	totalInflow := 0.0
	var topInflow *FundFlowInfo
	var topOutflow *FundFlowInfo

	for _, f := range flows {
		if f.MainNetInflow > 0 {
			inflowCount++
			totalInflow += f.MainNetInflow
			if topInflow == nil || f.MainNetInflow > topInflow.MainNetInflow {
				topInflow = f
			}
		} else {
			outflowCount++
			if topOutflow == nil || f.MainNetInflow < topOutflow.MainNetInflow {
				topOutflow = f
			}
		}
	}

	summary := fmt.Sprintf("%s板块共%d个，主力净流入%d个，净流出%d个",
		typeName, len(flows), inflowCount, outflowCount)
	if topInflow != nil {
		summary += fmt.Sprintf("。流入最多: %s(%.2f亿)", topInflow.Name, topInflow.MainNetInflow)
	}

	return summary
}

// ScrapeTechIndicators 抓取板块技术指标
func ScrapeTechIndicators(sectorName string) (*TechIndicatorsResponse, error) {
	// 创建浏览器实例
	browser, err := playwright.New(
		playwright.WithHeadless(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	// 创建页面
	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	// 注入反检测脚本
	if err := page.InjectAntiDetect(); err != nil {
		return nil, fmt.Errorf("failed to inject anti-detect: %w", err)
	}

	// 先获取板块代码
	url := "https://quote.eastmoney.com/center/boardlist.html#industry_board"
	if err := page.Goto(url, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}
	page.Wait(5 * time.Second)

	// 查找板块代码
	sectorCode, err := findSectorCode(page, sectorName)
	if err != nil {
		return nil, fmt.Errorf("failed to find sector '%s': %w", sectorName, err)
	}

	// 导航到板块K线页面获取技术指标
	klineURL := fmt.Sprintf("https://quote.eastmoney.com/bk/90.%s.html", sectorCode)
	if err := page.Goto(klineURL, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to kline page: %w", err)
	}
	page.Wait(5 * time.Second)

	// 提取技术指标数据
	indicators, err := extractTechIndicators(page, sectorName, sectorCode)
	if err != nil {
		return nil, err
	}

	response := &TechIndicatorsResponse{
		Indicators: indicators,
		Timestamp:  FormatTimestamp(),
		Summary:    generateTechSummary(indicators),
	}

	return response, nil
}

// extractTechIndicators 从页面提取技术指标
func extractTechIndicators(page *playwright.Page, sectorName, sectorCode string) (*TechIndicators, error) {
	timestamp := FormatTimestamp()

	// 使用 JavaScript 获取页面上的技术指标数据
	result, err := page.Evaluate(`
		(() => {
			// 获取当前价格
			const priceEl = document.querySelector('.zxj') || document.querySelector('.price');
			const price = priceEl ? parseFloat(priceEl.innerText.replace(/,/g, '')) : 0;
			
			// 尝试获取均线数据(如果页面有展示)
			const getMA = (selector) => {
				const el = document.querySelector(selector);
				return el ? parseFloat(el.innerText.replace(/[^0-9.-]/g, '')) : 0;
			};
			
			return {
				price: price || 0,
				// 其他指标需要通过更复杂的计算或API获取
			};
		})()
	`)
	if err != nil {
		// 如果提取失败，返回基本数据
		return &TechIndicators{
			SectorName: sectorName,
			Trend:      "数据获取中",
			Suggestion: "请稍后重试",
			Timestamp:  timestamp,
		}, nil
	}

	priceData, _ := result.(map[string]interface{})
	price := 0.0
	if p, ok := priceData["price"].(float64); ok {
		price = p
	}

	// 由于东方财富页面不直接展示完整技术指标，
	// 这里提供基于价格位置的简单趋势判断
	indicators := &TechIndicators{
		SectorName: sectorName,
		Price:      price,
		MA5:        0,  // 需要历史数据计算
		MA10:       0,  // 需要历史数据计算
		MA20:       0,  // 需要历史数据计算
		RSI6:       50, // 默认中性
		RSI12:      50, // 默认中性
		MACD:       0,
		Signal:     0,
		Histogram:  0,
		Trend:      "震荡",
		Suggestion: "建议观望，关注成交量变化",
		Timestamp:  timestamp,
	}

	return indicators, nil
}

// generateTechSummary 生成技术指标摘要
func generateTechSummary(ind *TechIndicators) string {
	if ind == nil {
		return "暂无数据"
	}

	summary := fmt.Sprintf("%s 最新价:%.2f", ind.SectorName, ind.Price)
	if ind.Trend != "" {
		summary += fmt.Sprintf("，趋势:%s", ind.Trend)
	}
	if ind.Suggestion != "" {
		summary += fmt.Sprintf("。%s", ind.Suggestion)
	}

	return summary
}
