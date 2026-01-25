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

// ScrapeHotSectors 抓取热门板块(涨跌幅前列)
func ScrapeHotSectors(limit int) (*HotSectorsResponse, error) {
	// 抓取行业板块
	industrySectors, err := ScrapeSectors(SectorTypeIndustry, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape industry sectors: %w", err)
	}

	// 按涨跌幅排序
	sectors := industrySectors.Sectors

	// 涨幅前列
	sort.Slice(sectors, func(i, j int) bool {
		return sectors[i].ChangeRate > sectors[j].ChangeRate
	})
	topRising := make([]*SectorInfo, 0, limit)
	for i := 0; i < len(sectors) && i < limit; i++ {
		if sectors[i].ChangeRate > 0 {
			topRising = append(topRising, sectors[i])
		}
	}

	// 跌幅前列
	sort.Slice(sectors, func(i, j int) bool {
		return sectors[i].ChangeRate < sectors[j].ChangeRate
	})
	topFalling := make([]*SectorInfo, 0, limit)
	for i := 0; i < len(sectors) && i < limit; i++ {
		if sectors[i].ChangeRate < 0 {
			topFalling = append(topFalling, sectors[i])
		}
	}

	response := &HotSectorsResponse{
		TopRising:  topRising,
		TopFalling: topFalling,
		Timestamp:  FormatTimestamp(),
		Summary:    generateHotSummary(topRising, topFalling),
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
	var parts []string

	if len(rising) > 0 {
		names := make([]string, 0, 3)
		for i := 0; i < len(rising) && i < 3; i++ {
			names = append(names, fmt.Sprintf("%s(+%.2f%%)", rising[i].Name, rising[i].ChangeRate))
		}
		parts = append(parts, "涨幅前三: "+strings.Join(names, "、"))
	}

	if len(falling) > 0 {
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
