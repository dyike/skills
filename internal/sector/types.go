package sector

import "time"

// SectorType 板块类型
type SectorType string

const (
	SectorTypeIndustry SectorType = "industry" // 行业板块
	SectorTypeConcept  SectorType = "concept"  // 概念板块
)

// SectorInfo 板块信息
type SectorInfo struct {
	Code        string  `json:"code"`         // 板块代码
	Name        string  `json:"name"`         // 板块名称
	Price       float64 `json:"price"`        // 最新价/指数
	Change      float64 `json:"change"`       // 涨跌额
	ChangeRate  float64 `json:"change_rate"`  // 涨跌幅%
	Volume      int64   `json:"volume"`       // 成交量(手)
	Amount      float64 `json:"amount"`       // 成交额(亿)
	LeaderStock string  `json:"leader_stock"` // 领涨股
	LeaderRate  float64 `json:"leader_rate"`  // 领涨股涨幅%
	RiseCount   int     `json:"rise_count"`   // 上涨家数
	FallCount   int     `json:"fall_count"`   // 下跌家数
	Timestamp   string  `json:"timestamp"`    // 数据时间
}

// SectorListResponse 板块列表响应
type SectorListResponse struct {
	Type      SectorType    `json:"type"`      // 板块类型
	Count     int           `json:"count"`     // 板块数量
	Sectors   []*SectorInfo `json:"sectors"`   // 板块列表
	Timestamp string        `json:"timestamp"` // 抓取时间
	Summary   string        `json:"summary"`   // 摘要信息
}

// HotSectorsResponse 热门板块响应
type HotSectorsResponse struct {
	TopRising  []*SectorInfo `json:"top_rising"`  // 涨幅前列
	TopFalling []*SectorInfo `json:"top_falling"` // 跌幅前列
	Timestamp  string        `json:"timestamp"`   // 抓取时间
	Summary    string        `json:"summary"`     // 摘要信息
}

// SectorURLs 东方财富板块数据 URL
var SectorURLs = map[SectorType]string{
	SectorTypeIndustry: "https://quote.eastmoney.com/center/boardlist.html#industry_board",
	SectorTypeConcept:  "https://quote.eastmoney.com/center/boardlist.html#concept_board",
}

// FormatTimestamp 格式化当前时间戳
func FormatTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// StockInfo 个股信息
type StockInfo struct {
	Code       string  `json:"code"`        // 股票代码
	Name       string  `json:"name"`        // 股票名称
	Price      float64 `json:"price"`       // 最新价
	Change     float64 `json:"change"`      // 涨跌额
	ChangeRate float64 `json:"change_rate"` // 涨跌幅%
	Volume     int64   `json:"volume"`      // 成交量(手)
	Amount     float64 `json:"amount"`      // 成交额(亿)
	Turnover   float64 `json:"turnover"`    // 换手率%
	PE         float64 `json:"pe"`          // 市盈率
	Timestamp  string  `json:"timestamp"`   // 数据时间
}

// SectorStocksResponse 板块个股响应
type SectorStocksResponse struct {
	SectorName string       `json:"sector_name"` // 板块名称
	SectorCode string       `json:"sector_code"` // 板块代码
	Count      int          `json:"count"`       // 个股数量
	Stocks     []*StockInfo `json:"stocks"`      // 个股列表
	Timestamp  string       `json:"timestamp"`   // 抓取时间
	Summary    string       `json:"summary"`     // 摘要信息
}
