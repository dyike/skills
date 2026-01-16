package contentcreator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// 全局配置
var cfg struct {
	limit      int
	outputDir  string
	format     string
	apiKey     string
	model      string
	useBrowser bool
	inputFile  string
	topic      string
	style      string // stanley, defou, combo, all
	imageMode  string // auto, placeholder, none
	withImages bool   // 是否生成配图
}

// NewCmd 创建根命令
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "content-creator",
		Short: "智能内容创作工作流工具",
		Long:  `基于 Defou 方法论的智能内容创作助手。支持热点挖掘、内容生成、爆款验证。`,
	}

	// 全局 flags
	root.PersistentFlags().IntVarP(&cfg.limit, "limit", "l", 20, "抓取热榜条目数量")
	root.PersistentFlags().StringVarP(&cfg.outputDir, "output-dir", "o", "./outputs", "输出目录")
	root.PersistentFlags().StringVarP(&cfg.format, "format", "f", "markdown", "输出格式: markdown, json")
	root.PersistentFlags().StringVar(&cfg.apiKey, "api-key", "", "Claude API Key (默认从环境变量读取)")
	root.PersistentFlags().StringVar(&cfg.model, "model", "", "Claude 模型 (默认 claude-sonnet-4-5)")
	root.PersistentFlags().BoolVar(&cfg.useBrowser, "use-browser", false, "使用浏览器模式抓取 (处理 JS 渲染)")

	// 子命令
	root.AddCommand(
		newTopHubCmd(),
		newGenerateCmd(),
		newVerifyCmd(),
		newAutoCmd(),
		newImageCmd(),
	)

	return root
}

// newTopHubCmd 热点挖掘命令
func newTopHubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tophub",
		Short: "抓取并分析 TopHub 热榜",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(os.Stderr, "🔍 正在抓取 TopHub 热榜 (limit: %d)...\n", cfg.limit)

			var topics []HotTopic
			var err error

			if cfg.useBrowser {
				topics, err = ScrapeTopHubWithBrowser(cfg.limit)
			} else {
				topics, err = ScrapeTopHub(cfg.limit)
			}

			if err != nil {
				return fmt.Errorf("抓取失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ 成功抓取 %d 条热点\n", len(topics))

			// 去重
			topics = DeduplicateTopics(topics)
			fmt.Fprintf(os.Stderr, "🔄 去重后剩余 %d 条\n", len(topics))

			// 调用 Claude 分析
			fmt.Fprintf(os.Stderr, "🤖 正在使用 AI 分析流量潜力...\n")

			client := NewClaudeClient()
			if cfg.apiKey != "" {
				client.APIKey = cfg.apiKey
			}
			if cfg.model != "" {
				client.Model = cfg.model
			}

			analysis, err := client.AnalyzeTrends(topics, cfg.limit)
			if err != nil {
				return fmt.Errorf("AI 分析失败: %w", err)
			}

			// 保存结果
			if err := saveAnalysis(analysis); err != nil {
				return fmt.Errorf("保存结果失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ 分析完成\n")
			return nil
		},
	}
}

// newGenerateCmd 内容生成命令
func newGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "生成内容（Stanley/Defou/Combo 风格）",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := &GenerateRequest{
				Topic: cfg.topic,
				Style: cfg.style,
			}

			// 读取输入文件
			if cfg.inputFile != "" {
				content, err := os.ReadFile(cfg.inputFile)
				if err != nil {
					return fmt.Errorf("读取文件失败: %w", err)
				}
				req.RawContent = string(content)
			}

			if req.Topic == "" && req.RawContent == "" {
				return fmt.Errorf("请提供 --topic 或 --input 参数")
			}

			fmt.Fprintf(os.Stderr, "🎨 正在生成内容 (风格: %s)...\n", cfg.style)

			client := NewClaudeClient()
			if cfg.apiKey != "" {
				client.APIKey = cfg.apiKey
			}
			if cfg.model != "" {
				client.Model = cfg.model
			}

			// 读取 Prompt 模板
			stylePrompt, err := loadPrompt(cfg.style)
			if err != nil {
				return fmt.Errorf("加载 Prompt 失败: %w", err)
			}

			// 生成内容
			content, err := client.GenerateContent(req, stylePrompt)
			if err != nil {
				return fmt.Errorf("生成失败: %w", err)
			}

			// 保存结果
			if _, err := saveGenerated(content, cfg.style); err != nil {
				return fmt.Errorf("保存失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ 生成完成\n")
			return nil
		},
	}

	cmd.Flags().StringVarP(&cfg.inputFile, "input", "i", "", "输入文件路径（草稿）")
	cmd.Flags().StringVarP(&cfg.topic, "topic", "t", "", "话题")
	cmd.Flags().StringVarP(&cfg.style, "style", "s", "combo", "风格: stanley, defou, combo, all")

	return cmd
}

// newVerifyCmd 验证命令
func newVerifyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "验证并优化内容",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.inputFile == "" {
				return fmt.Errorf("请提供 --input 参数")
			}

			content, err := os.ReadFile(cfg.inputFile)
			if err != nil {
				return fmt.Errorf("读取文件失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "🩺 正在验证内容...\n")

			client := NewClaudeClient()
			if cfg.apiKey != "" {
				client.APIKey = cfg.apiKey
			}
			if cfg.model != "" {
				client.Model = cfg.model
			}

			// 读取验证 Prompt
			verifyPrompt, err := loadPrompt("verify")
			if err != nil {
				return fmt.Errorf("加载验证 Prompt 失败: %w", err)
			}

			// 验证内容
			result, err := client.VerifyContent(string(content), verifyPrompt)
			if err != nil {
				return fmt.Errorf("验证失败: %w", err)
			}

			// 保存结果
			if err := saveVerified(result); err != nil {
				return fmt.Errorf("保存失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ 验证完成\n")
			return nil
		},
	}

	cmd.Flags().StringVarP(&cfg.inputFile, "input", "i", "", "输入文件路径")

	return cmd
}

// newImageCmd 图片生成命令
func newImageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "为文章生成配图",
		RunE: func(cmd *cobra.Command, args []string) error {
			if cfg.inputFile == "" {
				return fmt.Errorf("请提供 --input 参数指定文章文件")
			}

			content, err := os.ReadFile(cfg.inputFile)
			if err != nil {
				return fmt.Errorf("读取文件失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "🎨 正在为文章生成配图...\\n")
			fmt.Fprintf(os.Stderr, "📄 文章文件: %s\\n", cfg.inputFile)
			fmt.Fprintf(os.Stderr, "🔧 配图模式: %s\\n\\n", cfg.imageMode)

			imageGen := NewImageGenerator(cfg.outputDir)
			if cfg.imageMode == "" {
				cfg.imageMode = "auto"
			}

			contentWithImages, imagePaths, err := imageGen.GenerateImagesForArticle(string(content), cfg.imageMode)
			if err != nil {
				return fmt.Errorf("生成配图失败: %w", err)
			}

			if len(imagePaths) == 0 {
				fmt.Fprintf(os.Stderr, "ℹ️ 未生成任何配图\\n")
				return nil
			}

			// 保存带配图的文章
			outputPath, err := saveWithImages(contentWithImages)
			if err != nil {
				return fmt.Errorf("保存失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\\n✅ 配图生成完成！\\n")
			fmt.Fprintf(os.Stderr, "📸 生成 %d 张配图\\n", len(imagePaths))
			fmt.Fprintf(os.Stderr, "💾 文章已保存: %s\\n", outputPath)

			return nil
		},
	}

	cmd.Flags().StringVarP(&cfg.inputFile, "input", "i", "", "输入文章文件路径")
	cmd.Flags().StringVar(&cfg.imageMode, "mode", "auto", "配图模式: auto(AI自动分析), placeholder(识别占位符)")

	return cmd
}

// newAutoCmd 全自动命令
func newAutoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto",
		Short: "全自动模式：抓取 -> 生成 -> 配图 -> 验证",
		RunE: func(cmd *cobra.Command, args []string) error {
			totalSteps := 3
			if cfg.withImages {
				totalSteps = 4
			}

			fmt.Fprintf(os.Stderr, "🚀 启动全自动内容创作流程...\\n\\n")

			// Step 1: 抓取热点
			fmt.Fprintf(os.Stderr, "【步骤 1/%d】抓取热点\\n", totalSteps)
			var topics []HotTopic
			var err error

			if cfg.useBrowser {
				topics, err = ScrapeTopHubWithBrowser(cfg.limit)
			} else {
				topics, err = ScrapeTopHub(cfg.limit)
			}

			if err != nil {
				return fmt.Errorf("抓取失败: %w", err)
			}

			topics = DeduplicateTopics(topics)
			fmt.Fprintf(os.Stderr, "✅ 抓取成功: %d 条热点\\n\\n", len(topics))

			// Step 2: 生成内容
			fmt.Fprintf(os.Stderr, "【步骤 2/%d】生成内容\\n", totalSteps)

			client := NewClaudeClient()
			if cfg.apiKey != "" {
				client.APIKey = cfg.apiKey
			}
			if cfg.model != "" {
				client.Model = cfg.model
			}

			// 分析并选择最佳话题
			analysis, err := client.AnalyzeTrends(topics, 10)
			if err != nil {
				return fmt.Errorf("分析失败: %w", err)
			}

			// 保存分析结果
			saveAnalysis(analysis)

			// 使用推荐的第一个话题生成内容
			var selectedTopic string
			if len(analysis.Recommendations) > 0 {
				selectedTopic = analysis.Recommendations[0].Topic
			} else if len(topics) > 0 {
				selectedTopic = topics[0].Title
			} else {
				return fmt.Errorf("没有可用的话题")
			}

			fmt.Fprintf(os.Stderr, "📝 选定话题: %s\\n", selectedTopic)

			req := &GenerateRequest{
				Topic:   selectedTopic,
				Style:   "combo",
				Sources: topics[:min(5, len(topics))],
			}

			stylePrompt, err := loadPrompt("combo")
			if err != nil {
				return fmt.Errorf("加载 Prompt 失败: %w", err)
			}

			content, err := client.GenerateContent(req, stylePrompt)
			if err != nil {
				return fmt.Errorf("生成失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ 生成完成\\n\\n")

			// Step 3 (可选): 生成配图
			currentStep := 3
			if cfg.withImages {
				fmt.Fprintf(os.Stderr, "【步骤 %d/%d】生成配图\\n", currentStep, totalSteps)

				imageGen := NewImageGenerator(cfg.outputDir)
				imageMode := cfg.imageMode
				if imageMode == "" {
					imageMode = "auto"
				}

				contentWithImages, imagePaths, err := imageGen.GenerateImagesForArticle(content, imageMode)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠️ 配图生成失败: %v，继续使用原文\\n", err)
				} else if len(imagePaths) > 0 {
					content = contentWithImages
					fmt.Fprintf(os.Stderr, "✅ 生成 %d 张配图\\n\\n", len(imagePaths))
				} else {
					fmt.Fprintf(os.Stderr, "ℹ️ 无需生成配图\\n\\n")
				}
				currentStep++
			}

			// 保存生成的内容
			generatedFile, err := saveGenerated(content, "combo")
			if err != nil {
				return fmt.Errorf("保存失败: %w", err)
			}

			// Step: 验证优化
			fmt.Fprintf(os.Stderr, "【步骤 %d/%d】验证优化\\n", currentStep, totalSteps)

			verifyPrompt, err := loadPrompt("verify")
			if err != nil {
				return fmt.Errorf("加载验证 Prompt 失败: %w", err)
			}

			result, err := client.VerifyContent(content, verifyPrompt)
			if err != nil {
				return fmt.Errorf("验证失败: %w", err)
			}

			if err := saveVerified(result); err != nil {
				return fmt.Errorf("保存失败: %w", err)
			}

			fmt.Fprintf(os.Stderr, "✅ 验证完成\\n\\n")
			fmt.Fprintf(os.Stderr, "🎉 全流程完成！生成文件: %s\\n", generatedFile)

			return nil
		},
	}

	cmd.Flags().BoolVar(&cfg.withImages, "with-images", false, "是否生成配图")
	cmd.Flags().StringVar(&cfg.imageMode, "image-mode", "auto", "配图模式: auto, placeholder")

	return cmd
}

// 辅助函数

func saveAnalysis(analysis *TrendAnalysis) error {
	// 确保输出目录存在
	trendsDir := filepath.Join(cfg.outputDir, "trends")
	if err := os.MkdirAll(trendsDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("tophub_analysis_%s.md", timestamp)
	outputPath := filepath.Join(trendsDir, filename)

	// 生成 Markdown
	var sb strings.Builder
	sb.WriteString("# TopHub 热点分析报告\n\n")
	sb.WriteString(fmt.Sprintf("生成时间：%s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if analysis.Summary != "" {
		sb.WriteString("## 趋势总结\n\n")
		sb.WriteString(analysis.Summary)
		sb.WriteString("\n\n")
	}

	if len(analysis.Recommendations) > 0 {
		sb.WriteString("## 推荐选题\n\n")
		for i, rec := range analysis.Recommendations {
			sb.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, rec.Topic))
			sb.WriteString(fmt.Sprintf("**潜力评分**: %d/100\n\n", rec.Potential))
			sb.WriteString(fmt.Sprintf("**推荐理由**: %s\n\n", rec.Reason))
			sb.WriteString(fmt.Sprintf("**切入角度**: %s\n\n", rec.Angle))
			sb.WriteString("---\n\n")
		}
	}

	sb.WriteString("## 全部热点\n\n")
	for i, topic := range analysis.Topics {
		sb.WriteString(fmt.Sprintf("%d. [%s](%s) - %s\n", i+1, topic.Title, topic.Link, topic.Hot))
	}

	if err := os.WriteFile(outputPath, []byte(sb.String()), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "💾 分析报告已保存: %s\n", outputPath)

	// 同时保存 JSON
	jsonFile := filepath.Join(trendsDir, fmt.Sprintf("tophub_data_%s.json", timestamp))
	data, _ := json.MarshalIndent(analysis, "", "  ")
	os.WriteFile(jsonFile, data, 0644)

	return nil
}

func saveGenerated(content, style string) (string, error) {
	genDir := filepath.Join(cfg.outputDir, "generated")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.md", style, timestamp)
	outputPath := filepath.Join(genDir, filename)

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", err
	}

	fmt.Fprintf(os.Stderr, "💾 生成内容已保存: %s\n", outputPath)
	return outputPath, nil
}

func saveVerified(result string) error {
	verifiedDir := filepath.Join(cfg.outputDir, "verified")
	if err := os.MkdirAll(verifiedDir, 0755); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("verified_%s.md", timestamp)
	outputPath := filepath.Join(verifiedDir, filename)

	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "💾 验证报告已保存: %s\n", outputPath)
	return nil
}

func saveWithImages(content string) (string, error) {
	genDir := filepath.Join(cfg.outputDir, "generated")
	if err := os.MkdirAll(genDir, 0755); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("article_with_images_%s.md", timestamp)
	outputPath := filepath.Join(genDir, filename)

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", err
	}

	return outputPath, nil
}

func loadPrompt(style string) (string, error) {
	// 这里简化处理，实际可以从 skill/content-creator/references/ 读取
	switch style {
	case "combo":
		return loadPromptFile("combo_style.md")
	case "stanley":
		return "Stanley 风格 Prompt（极致爆款）", nil
	case "defou":
		return "Defou 风格 Prompt（深度认知）", nil
	case "verify":
		return loadPromptFile("viral_verification.md")
	default:
		return loadPromptFile("combo_style.md")
	}
}

func loadPromptFile(filename string) (string, error) {
	// 尝试从多个位置读取
	paths := []string{
		filepath.Join("skill/content-creator/references", filename),
		filepath.Join("../skill/content-creator/references", filename),
		filepath.Join("../../skill/content-creator/references", filename),
	}

	for _, path := range paths {
		if content, err := os.ReadFile(path); err == nil {
			return string(content), nil
		}
	}

	// 如果找不到，返回简化版本
	return fmt.Sprintf("使用 %s 风格创作内容", filename), nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
