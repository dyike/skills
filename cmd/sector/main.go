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
		Long:  "获取涨跌幅前列的热门板块",
		Run:   handleHot,
	}
	hotCmd.Flags().IntP("limit", "l", 10, "返回数量限制")

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

	rootCmd.AddCommand(listCmd, hotCmd, stocksCmd)

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

	result, err := sector.ScrapeHotSectors(limit)
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
