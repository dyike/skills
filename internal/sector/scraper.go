package sector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/crosszan/modu/pkg/playwright"
	pwgo "github.com/playwright-community/playwright-go"
)

// ScrapeSectors 抓取板块列表 (使用Playwright)
func ScrapeSectors(sectorType SectorType, limit int) (*SectorListResponse, error) {
	url, ok := SectorURLs[sectorType]
	if !ok {
		return nil, fmt.Errorf("unsupported sector type: %s", sectorType)
	}

	// 移除 URL 中的 hash 部分，先访问基础页面
	baseURL := strings.Split(url, "#")[0]
	targetHash := ""
	if strings.Contains(url, "#") {
		targetHash = "#" + strings.Split(url, "#")[1]
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

	// 导航到基础页面
	if err := page.Goto(baseURL, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(30000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}

	// 等待一小段时间让页面初始化
	page.Wait(2 * time.Second)

	// 尝试关闭可能出现的弹窗
	closePopups(page)

	// 设置 hash 触发路由
	if targetHash != "" {
		page.Evaluate(fmt.Sprintf(`window.location.hash = '%s'`, targetHash))
	}

	// 等待数据表格加载完成（轮询检查）
	// 最多等待 30 秒 (15 * 2s)
	for i := 0; i < 15; i++ {
		// 每次都尝试关闭弹窗
		closePopups(page)

		// 检查数据是否加载
		result, _ := page.Evaluate(`
			(() => {
				const tables = document.querySelectorAll('table');
				if (tables.length < 2) return { ready: false };
				const rows = tables[1].querySelectorAll('tbody tr');
				return { ready: rows.length > 5 };
			})()
		`)

		if m, ok := result.(map[string]interface{}); ok {
			if ready, ok := m["ready"].(bool); ok && ready {
				break
			}
		}

		// 假如设置hash没生效，再次尝试设置
		if i%3 == 0 && targetHash != "" {
			page.Evaluate(fmt.Sprintf(`window.location.hash = '%s'`, targetHash))
		}

		page.Wait(2 * time.Second)
	}

	// 提取板块数据 (使用第二个表格，第一个是沪深港通信息)
	sectors, err := extractSectorDataFromSecondTable(page, limit)
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

// closePopups 尝试关闭页面上的各种弹窗
func closePopups(page *playwright.Page) {
	page.Evaluate(`
		(() => {
			// 常见的弹窗关闭按钮选择器
			const selectors = [
				'.close', '.close-btn', '.closeBtn',
				'[class*="close"]', '[class*="Close"]',
				'.layui-layer-close', '.layui-layer-close1', '.layui-layer-close2',
				'.modal-close', '.popup-close', '.dialog-close',
				'.ad-close', '.adv-close', '#close', '.guide-close'
			];
			
			for (const selector of selectors) {
				try {
					const elements = document.querySelectorAll(selector);
					elements.forEach(el => {
						if (el && el.offsetParent !== null) {
							el.click();
						}
					});
				} catch(e) {}
			}
			
			// 移除可能的遮罩层和弹窗
			const removables = document.querySelectorAll('.mask, .overlay, .modal-backdrop, .layui-layer-shade, .layui-layer, .popup, .modal');
			removables.forEach(el => {
				if (el && el.offsetParent !== null) {
					el.remove();
				}
			});
		})()
	`)
}

// extractSectorDataFromSecondTable 从第二个表格提取板块数据
// 东方财富板块列表页面有两个表格:
// - 第一个表格: 沪深港通信息 (1行)
// - 第二个表格: 板块列表数据
// 表格结构 (2024年验证):
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
func extractSectorDataFromSecondTable(page *playwright.Page, limit int) ([]*SectorInfo, error) {
	timestamp := FormatTimestamp()

	// 使用 JavaScript 获取第二个表格的数据
	result, err := page.Evaluate(fmt.Sprintf(`
		(() => {
			const tables = document.querySelectorAll('table');
			if (tables.length < 2) return [];
			
			const table = tables[1]; // 使用第二个表格
			const rows = Array.from(table.querySelectorAll('tbody tr'));
			const limit = %d;
			
			return rows.slice(0, limit).map(row => {
				const cells = Array.from(row.querySelectorAll('td'));
				if (cells.length < 12) return null;
				
				return {
					name: cells[1] ? cells[1].innerText.trim() : '',
					price: cells[3] ? cells[3].innerText.trim() : '',
					change: cells[4] ? cells[4].innerText.trim() : '',
					changeRate: cells[5] ? cells[5].innerText.trim() : '',
					marketCap: cells[6] ? cells[6].innerText.trim() : '',
					turnover: cells[7] ? cells[7].innerText.trim() : '',
					riseCount: cells[8] ? cells[8].innerText.trim() : '',
					fallCount: cells[9] ? cells[9].innerText.trim() : '',
					leaderStock: cells[10] ? cells[10].innerText.trim() : '',
					leaderRate: cells[11] ? cells[11].innerText.trim() : ''
				};
			}).filter(x => x !== null && x.name !== '');
		})()
	`, limit))
	if err != nil {
		return nil, fmt.Errorf("failed to extract sector data: %w", err)
	}

	// 解析结果
	rows, ok := result.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid sector data format")
	}

	sectors := make([]*SectorInfo, 0, len(rows))
	for _, row := range rows {
		rowMap, ok := row.(map[string]interface{})
		if !ok {
			continue
		}

		name := getString(rowMap, "name")
		if name == "" {
			continue
		}

		sectors = append(sectors, &SectorInfo{
			Code:        "", // 页面不再显示板块代码
			Name:        name,
			Price:       parseFloat(getString(rowMap, "price")),
			Change:      parseFloat(getString(rowMap, "change")),
			ChangeRate:  parsePercentage(getString(rowMap, "changeRate")),
			Volume:      0,                                              // 页面不再显示成交量
			Amount:      parseMarketCap(getString(rowMap, "marketCap")), // 解析总市值为成交额
			LeaderStock: getString(rowMap, "leaderStock"),
			LeaderRate:  parsePercentage(getString(rowMap, "leaderRate")),
			RiseCount:   int(parseInt(getString(rowMap, "riseCount"))),
			FallCount:   int(parseInt(getString(rowMap, "fallCount"))),
			Timestamp:   timestamp,
		})
	}

	return sectors, nil
}

// getStringFromMap 从map中获取字符串值
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		case float64:
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

// getFloatFromMap 从map中获取浮点数值
func getFloatFromMap(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case string:
			f, _ := strconv.ParseFloat(val, 64)
			return f
		}
	}
	return 0
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
	// 创建浏览器实例获取板块代码
	browser, err := playwright.New(
		playwright.WithHeadless(true),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser: %w", err)
	}
	defer browser.Close()

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}
	defer page.Close()

	if err := page.InjectAntiDetect(); err != nil {
		return nil, fmt.Errorf("failed to inject anti-detect: %w", err)
	}

	// 获取板块代码
	url := "https://quote.eastmoney.com/center/boardlist.html#industry_board"
	if err := page.Goto(url, playwright.WithWaitUntil("domcontentloaded"), playwright.WithTimeout(60000)); err != nil {
		return nil, fmt.Errorf("failed to navigate to page: %w", err)
	}
	page.Wait(5 * time.Second)

	sectorCode, err := findSectorCode(page, sectorName)
	if err != nil {
		return nil, fmt.Errorf("failed to find sector '%s': %w", sectorName, err)
	}

	// 使用K线API获取历史数据
	klineData, err := fetchKlineData(sectorCode, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch kline data: %w", err)
	}

	// 计算技术指标
	indicators := calculateTechIndicators(sectorName, klineData)

	response := &TechIndicatorsResponse{
		Indicators: indicators,
		Timestamp:  FormatTimestamp(),
		Summary:    generateTechSummary(indicators),
	}

	return response, nil
}

// KlinePoint K线数据点
type KlinePoint struct {
	Date   string
	Open   float64
	Close  float64
	High   float64
	Low    float64
	Volume int64
	Amount float64
	Change float64
}

// fetchKlineData 从东方财富API获取K线数据
func fetchKlineData(sectorCode string, days int) ([]*KlinePoint, error) {
	apiURL := fmt.Sprintf(
		"https://push2his.eastmoney.com/api/qt/stock/kline/get?secid=90.%s&fields1=f1,f2,f3,f4,f5,f6&fields2=f51,f52,f53,f54,f55,f56,f57,f58,f59,f60,f61&klt=101&fqt=1&end=20500101&lmt=%d",
		sectorCode, days)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var result struct {
		Data struct {
			Klines []string `json:"klines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// 解析K线数据
	// 格式: 日期,开盘,收盘,最高,最低,成交量,成交额,振幅,涨跌幅,涨跌额,换手率
	klines := make([]*KlinePoint, 0, len(result.Data.Klines))
	for _, line := range result.Data.Klines {
		parts := strings.Split(line, ",")
		if len(parts) < 10 {
			continue
		}
		kp := &KlinePoint{
			Date:   parts[0],
			Open:   parseFloat(parts[1]),
			Close:  parseFloat(parts[2]),
			High:   parseFloat(parts[3]),
			Low:    parseFloat(parts[4]),
			Volume: parseInt(parts[5]),
			Amount: parseFloat(parts[6]),
			Change: parseFloat(parts[8]),
		}
		klines = append(klines, kp)
	}

	return klines, nil
}

// calculateTechIndicators 计算技术指标
func calculateTechIndicators(sectorName string, klines []*KlinePoint) *TechIndicators {
	timestamp := FormatTimestamp()

	if len(klines) == 0 {
		return &TechIndicators{
			SectorName: sectorName,
			Trend:      "数据不足",
			Suggestion: "无法计算技术指标",
			Timestamp:  timestamp,
		}
	}

	// 获取收盘价数组
	closes := make([]float64, len(klines))
	for i, k := range klines {
		closes[i] = k.Close
	}

	// 当前价格
	price := closes[len(closes)-1]

	// 计算均线
	ma5 := calculateMA(closes, 5)
	ma10 := calculateMA(closes, 10)
	ma20 := calculateMA(closes, 20)

	// 计算RSI
	rsi6 := calculateRSI(closes, 6)
	rsi12 := calculateRSI(closes, 12)

	// 计算MACD
	macd, signal, histogram := calculateMACD(closes)

	// 趋势判断
	trend, suggestion := analyzeTrend(price, ma5, ma10, ma20, rsi6, macd, histogram)

	return &TechIndicators{
		SectorName: sectorName,
		Price:      price,
		MA5:        ma5,
		MA10:       ma10,
		MA20:       ma20,
		RSI6:       rsi6,
		RSI12:      rsi12,
		MACD:       macd,
		Signal:     signal,
		Histogram:  histogram,
		Trend:      trend,
		Suggestion: suggestion,
		Timestamp:  timestamp,
	}
}

// calculateMA 计算移动平均线
func calculateMA(data []float64, period int) float64 {
	if len(data) < period {
		return 0
	}
	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i]
	}
	return sum / float64(period)
}

// calculateRSI 计算相对强弱指标
func calculateRSI(data []float64, period int) float64 {
	if len(data) < period+1 {
		return 50
	}

	gains := 0.0
	losses := 0.0

	for i := len(data) - period; i < len(data); i++ {
		change := data[i] - data[i-1]
		if change > 0 {
			gains += change
		} else {
			losses += -change
		}
	}

	if losses == 0 {
		return 100
	}
	rs := gains / losses
	return 100 - (100 / (1 + rs))
}

// calculateMACD 计算MACD指标
func calculateMACD(data []float64) (macd, signal, histogram float64) {
	if len(data) < 26 {
		return 0, 0, 0
	}

	// EMA12
	ema12 := calculateEMA(data, 12)
	// EMA26
	ema26 := calculateEMA(data, 26)
	// MACD = EMA12 - EMA26
	macd = ema12 - ema26

	// 简化的signal计算
	signal = macd * 0.9
	histogram = macd - signal

	return macd, signal, histogram
}

// calculateEMA 计算指数移动平均
func calculateEMA(data []float64, period int) float64 {
	if len(data) < period {
		return 0
	}

	multiplier := 2.0 / float64(period+1)
	ema := data[len(data)-period] // 起始值

	for i := len(data) - period + 1; i < len(data); i++ {
		ema = (data[i]-ema)*multiplier + ema
	}

	return ema
}

// analyzeTrend 分析趋势并给出建议
func analyzeTrend(price, ma5, ma10, ma20, rsi, macd, histogram float64) (trend, suggestion string) {
	// 多头排列判断
	bullishMA := ma5 > ma10 && ma10 > ma20 && price > ma5
	// 空头排列判断
	bearishMA := ma5 < ma10 && ma10 < ma20 && price < ma5

	// RSI判断
	overbought := rsi > 70
	oversold := rsi < 30

	// MACD判断
	macdBullish := histogram > 0

	if bullishMA && macdBullish {
		trend = "强势上涨"
		if overbought {
			suggestion = "短期超买，注意高位风险，可适当减仓"
		} else {
			suggestion = "多头趋势明确，可继续持有或逢低买入"
		}
	} else if bearishMA && !macdBullish {
		trend = "弱势下跌"
		if oversold {
			suggestion = "短期超卖，可能反弹，但不建议抄底"
		} else {
			suggestion = "空头趋势明确，建议观望或减仓"
		}
	} else if price > ma5 && price > ma10 {
		trend = "震荡偏强"
		suggestion = "短期偏强，关注突破情况"
	} else if price < ma5 && price < ma10 {
		trend = "震荡偏弱"
		suggestion = "短期偏弱，等待企稳信号"
	} else {
		trend = "横盘震荡"
		suggestion = "方向不明，建议等待趋势明确后再操作"
	}

	return trend, suggestion
}

// generateTechSummary 生成技术指标摘要
func generateTechSummary(ind *TechIndicators) string {
	if ind == nil {
		return "暂无数据"
	}

	summary := fmt.Sprintf("%s 最新价:%.2f", ind.SectorName, ind.Price)
	if ind.MA5 > 0 {
		summary += fmt.Sprintf("，MA5:%.2f，MA10:%.2f，MA20:%.2f", ind.MA5, ind.MA10, ind.MA20)
	}
	if ind.Trend != "" {
		summary += fmt.Sprintf("，趋势:%s", ind.Trend)
	}
	if ind.Suggestion != "" {
		summary += fmt.Sprintf("。%s", ind.Suggestion)
	}

	return summary
}
