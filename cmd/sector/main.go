package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/dyike/skills/internal/sector"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sector",
		Short: "A股板块分析工具",
		Long:  "查看国内A股行业板块和概念板块的实时行情，获取热门板块推荐。",
	}

	// list 命令
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "获取板块列表",
		Long:  "获取行业板块或概念板块的实时行情列表",
		Run:   handleList,
	}
	listCmd.Flags().StringP("type", "t", "industry", "板块类型: industry(行业) 或 concept(概念)")
	listCmd.Flags().IntP("limit", "l", 20, "返回数量限制")

	// hot 命令
	hotCmd := &cobra.Command{
		Use:   "hot",
		Short: "获取热门板块",
		Long:  "获取热门板块，支持按涨跌幅或成交额排序",
		Run:   handleHot,
	}
	hotCmd.Flags().IntP("limit", "l", 10, "返回数量限制")
	hotCmd.Flags().StringP("sort", "s", "change", "排序方式: change(涨跌幅) 或 amount(成交额)")

	// stocks 命令
	stocksCmd := &cobra.Command{
		Use:   "stocks",
		Short: "获取板块内个股",
		Long:  "获取指定板块内的个股列表",
		Run:   handleStocks,
	}
	stocksCmd.Flags().StringP("name", "n", "", "板块名称 (必填，如: 光伏设备)")
	stocksCmd.Flags().IntP("limit", "l", 20, "返回数量限制")
	stocksCmd.MarkFlagRequired("name")

	// flow 命令
	flowCmd := &cobra.Command{
		Use:   "flow",
		Short: "获取板块资金流向",
		Long:  "获取行业或概念板块的资金流向数据",
		Run:   handleFlow,
	}
	flowCmd.Flags().StringP("type", "t", "industry", "板块类型: industry(行业) 或 concept(概念)")
	flowCmd.Flags().IntP("limit", "l", 20, "返回数量限制")

	// tech 命令
	techCmd := &cobra.Command{
		Use:   "tech",
		Short: "获取板块技术指标",
		Long:  "获取指定板块的技术指标分析(MA/RSI/MACD等)",
		Run:   handleTech,
	}
	techCmd.Flags().StringP("name", "n", "", "板块名称 (必填，如: 光伏设备)")
	techCmd.MarkFlagRequired("name")

	rootCmd.AddCommand(listCmd, hotCmd, stocksCmd, flowCmd, techCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleList(cmd *cobra.Command, args []string) {
	sectorType, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")

	var st sector.SectorType
	switch sectorType {
	case "industry":
		st = sector.SectorTypeIndustry
	case "concept":
		st = sector.SectorTypeConcept
	default:
		outputError(fmt.Sprintf("invalid sector type: %s, use 'industry' or 'concept'", sectorType))
		return
	}

	result, err := sector.ScrapeSectors(st, limit)
	if err != nil {
		outputError(fmt.Sprintf("failed to scrape sectors: %v", err))
		return
	}

	outputJSON(result)
}

func handleHot(cmd *cobra.Command, args []string) {
	limit, _ := cmd.Flags().GetInt("limit")
	sortBy, _ := cmd.Flags().GetString("sort")

	var st sector.SortType
	switch sortBy {
	case "amount":
		st = sector.SortByAmount
	default:
		st = sector.SortByChange
	}

	result, err := sector.ScrapeHotSectors(limit, st)
	if err != nil {
		outputError(fmt.Sprintf("failed to scrape hot sectors: %v", err))
		return
	}

	outputJSON(result)
}

func handleStocks(cmd *cobra.Command, args []string) {
	name, _ := cmd.Flags().GetString("name")
	limit, _ := cmd.Flags().GetInt("limit")

	if name == "" {
		outputError("sector name is required, use --name or -n flag")
		return
	}

	result, err := sector.ScrapeSectorStocks(name, limit)
	if err != nil {
		outputError(fmt.Sprintf("failed to scrape sector stocks: %v", err))
		return
	}

	outputJSON(result)
}

func handleFlow(cmd *cobra.Command, args []string) {
	sectorType, _ := cmd.Flags().GetString("type")
	limit, _ := cmd.Flags().GetInt("limit")

	var st sector.SectorType
	switch sectorType {
	case "industry":
		st = sector.SectorTypeIndustry
	case "concept":
		st = sector.SectorTypeConcept
	default:
		outputError(fmt.Sprintf("invalid sector type: %s, use 'industry' or 'concept'", sectorType))
		return
	}

	result, err := sector.ScrapeFundFlow(st, limit)
	if err != nil {
		outputError(fmt.Sprintf("failed to scrape fund flow: %v", err))
		return
	}

	outputJSON(result)
}

func handleTech(cmd *cobra.Command, args []string) {
	name, _ := cmd.Flags().GetString("name")

	if name == "" {
		outputError("sector name is required, use --name or -n flag")
		return
	}

	result, err := sector.ScrapeTechIndicators(name)
	if err != nil {
		outputError(fmt.Sprintf("failed to scrape tech indicators: %v", err))
		return
	}

	outputJSON(result)
}

func outputJSON(data interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}

func outputError(message string) {
	result := map[string]interface{}{
		"error":   true,
		"message": message,
	}
	outputJSON(result)
	os.Exit(1)
}
