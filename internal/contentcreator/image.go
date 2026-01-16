package contentcreator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/crosszan/modu/pkg/env"
	genimagerepo "github.com/crosszan/modu/repos/gen_image_repo"
	genimagevo "github.com/crosszan/modu/vo/gen_image_vo"
)

// ImageGenerator 图片生成器
type ImageGenerator struct {
	repo      genimagerepo.ImageGenRepo
	outputDir string
}

// NewImageGenerator 创建图片生成器
func NewImageGenerator(outputDir string) *ImageGenerator {
	baseURL := env.GetDefault("GEMINI_BASE_URL", env.GetDefault("IMAGE_API_BASE_URL", "https://generativelanguage.googleapis.com"))
	apiKey := env.GetDefault("GEMINI_API_KEY", env.Get("IMAGE_API_KEY"))

	return &ImageGenerator{
		repo:      genimagerepo.NewGeminiImageImpl(baseURL, apiKey),
		outputDir: outputDir,
	}
}

// ImagePlaceholder 图片占位符信息
type ImagePlaceholder struct {
	Index       int    // 占位符索引
	Description string // 图片描述
	Position    int    // 在文章中的位置
	FullMatch   string // 完整匹配文本
}

// GenerateImagesForArticle 为文章生成配图
// 支持两种模式:
// 1. auto: AI 分析文章内容，自动决定在哪些位置插入图片
// 2. placeholder: 识别文章中的 ![描述](placeholder) 格式，生成对应图片
func (g *ImageGenerator) GenerateImagesForArticle(article string, mode string) (string, []string, error) {
	switch mode {
	case "auto":
		return g.generateImagesAuto(article)
	case "placeholder":
		return g.generateImagesFromPlaceholders(article)
	default:
		return g.generateImagesFromPlaceholders(article)
	}
}

// generateImagesFromPlaceholders 从文章中的占位符生成图片
// 识别格式: ![图片描述](placeholder) 或 <!-- image: 图片描述 -->
func (g *ImageGenerator) generateImagesFromPlaceholders(article string) (string, []string, error) {
	placeholders := g.extractPlaceholders(article)
	if len(placeholders) == 0 {
		return article, nil, nil
	}

	imagePaths := make([]string, 0, len(placeholders))
	result := article

	for i, ph := range placeholders {
		fmt.Fprintf(os.Stderr, "🎨 生成图片 %d/%d: %s\n", i+1, len(placeholders), ph.Description)

		imagePath, err := g.generateAndSaveImage(ph.Description, i+1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ 图片生成失败: %v\n", err)
			continue
		}

		imagePaths = append(imagePaths, imagePath)

		// 替换占位符为实际图片路径
		newImageTag := fmt.Sprintf("![%s](%s)", ph.Description, imagePath)
		result = strings.Replace(result, ph.FullMatch, newImageTag, 1)
	}

	return result, imagePaths, nil
}

// generateImagesAuto AI 自动分析并生成配图
func (g *ImageGenerator) generateImagesAuto(article string) (string, []string, error) {
	// 使用 AI 分析文章，决定需要哪些配图
	client := NewClaudeClient()

	systemPrompt := `你是一个专业的内容配图助手。分析用户提供的文章，决定需要在哪些位置添加配图。

请输出 JSON 格式：
{
  "images": [
    {
      "position": "在第X段后",
      "description": "图片描述（用于 AI 生图）",
      "insert_after": "这段文字之后..."
    }
  ]
}

规则：
1. 每篇文章建议 2-4 张配图
2. 图片描述要具体、适合 AI 生成
3. 选择文章中的关键转折点或重要概念处
4. 描述应该是视觉化的、具体的场景`

	userPrompt := fmt.Sprintf("请分析这篇文章，决定配图位置和描述：\n\n%s", article)

	resp, err := client.Complete(systemPrompt, userPrompt, 2048)
	if err != nil {
		return article, nil, fmt.Errorf("AI 分析失败: %w", err)
	}

	// 解析 AI 响应
	type ImageSuggestion struct {
		Position    string `json:"position"`
		Description string `json:"description"`
		InsertAfter string `json:"insert_after"`
	}
	var suggestions struct {
		Images []ImageSuggestion `json:"images"`
	}

	if err := parseJSONResponse(resp, &suggestions); err != nil {
		return article, nil, fmt.Errorf("解析 AI 响应失败: %w", err)
	}

	if len(suggestions.Images) == 0 {
		return article, nil, nil
	}

	imagePaths := make([]string, 0)
	result := article

	for i, img := range suggestions.Images {
		fmt.Fprintf(os.Stderr, "🎨 生成图片 %d/%d: %s\n", i+1, len(suggestions.Images), img.Description)

		imagePath, err := g.generateAndSaveImage(img.Description, i+1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ 图片生成失败: %v\n", err)
			continue
		}

		imagePaths = append(imagePaths, imagePath)

		// 在指定位置插入图片
		if img.InsertAfter != "" {
			imageTag := fmt.Sprintf("\n\n![%s](%s)\n", img.Description, imagePath)
			result = strings.Replace(result, img.InsertAfter, img.InsertAfter+imageTag, 1)
		}
	}

	return result, imagePaths, nil
}

// extractPlaceholders 提取文章中的图片占位符
func (g *ImageGenerator) extractPlaceholders(article string) []ImagePlaceholder {
	placeholders := make([]ImagePlaceholder, 0)

	// 模式1: ![描述](placeholder) 或 ![描述]()
	pattern1 := regexp.MustCompile(`!\[([^\]]+)\]\((placeholder|)\)`)
	matches1 := pattern1.FindAllStringSubmatchIndex(article, -1)
	for i, match := range matches1 {
		if len(match) >= 4 {
			fullMatch := article[match[0]:match[1]]
			description := article[match[2]:match[3]]
			placeholders = append(placeholders, ImagePlaceholder{
				Index:       i + 1,
				Description: description,
				Position:    match[0],
				FullMatch:   fullMatch,
			})
		}
	}

	// 模式2: <!-- image: 描述 -->
	pattern2 := regexp.MustCompile(`<!--\s*image:\s*([^>]+)\s*-->`)
	matches2 := pattern2.FindAllStringSubmatchIndex(article, -1)
	for i, match := range matches2 {
		if len(match) >= 4 {
			fullMatch := article[match[0]:match[1]]
			description := strings.TrimSpace(article[match[2]:match[3]])
			placeholders = append(placeholders, ImagePlaceholder{
				Index:       len(placeholders) + i + 1,
				Description: description,
				Position:    match[0],
				FullMatch:   fullMatch,
			})
		}
	}

	return placeholders
}

// generateAndSaveImage 生成并保存单张图片
func (g *ImageGenerator) generateAndSaveImage(description string, index int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 增强 prompt，使其更适合生图
	enhancedPrompt := fmt.Sprintf("创作一张高质量的配图：%s。风格要求：现代、专业、适合文章配图使用。", description)

	req := &genimagevo.GenImageRequest{
		UserPrompt: enhancedPrompt,
	}

	resp, err := g.repo.Generate(ctx, req)
	if err != nil {
		return "", fmt.Errorf("生成图片失败: %w", err)
	}

	if len(resp.Images) == 0 {
		return "", fmt.Errorf("没有生成图片")
	}

	// 保存图片
	imageDir := filepath.Join(g.outputDir, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	ext := getExtensionFromMimeType(resp.Images[0].MimeType)
	filename := fmt.Sprintf("article_image_%s_%d%s", timestamp, index, ext)
	imagePath := filepath.Join(imageDir, filename)

	if err := os.WriteFile(imagePath, resp.Images[0].Data, 0644); err != nil {
		return "", fmt.Errorf("保存图片失败: %w", err)
	}

	fmt.Fprintf(os.Stderr, "💾 图片已保存: %s\n", imagePath)
	return imagePath, nil
}

// getExtensionFromMimeType 从 MIME 类型获取文件扩展名
func getExtensionFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".png"
	}
}

// parseJSONResponse 解析 JSON 响应，支持从 Markdown 代码块中提取
func parseJSONResponse(text string, v interface{}) error {
	text = strings.TrimSpace(text)

	// 如果是 Markdown 代码块，提取 JSON
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		jsonLines := make([]string, 0)
		inJSON := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				if inJSON {
					break
				}
				inJSON = true
				continue
			}
			if inJSON {
				jsonLines = append(jsonLines, line)
			}
		}
		text = strings.Join(jsonLines, "\n")
	}

	return json.Unmarshal([]byte(text), v)
}
