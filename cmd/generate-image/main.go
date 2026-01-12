package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	AntigravityBaseURL = "http://127.0.0.1:8045"
	APIKey             = "sk-5fec10a3ada64c0b808122ee2b971a5d"
)

// 请求结构
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type Part struct {
	Text string `json:"text"`
}

// 响应结构
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text       string      `json:"text,omitempty"`
				InlineData *InlineData `json:"inlineData,omitempty"`
			} `json:"parts"`
			Role string `json:"role"`
		} `json:"content"`
		FinishReason string `json:"finishReason"`
	} `json:"candidates"`
	ModelVersion  string        `json:"modelVersion"`
	ResponseID    string        `json:"responseId"`
	UsageMetadata UsageMetadata `json:"usageMetadata"`
}

type InlineData struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     string `json:"data"` // Base64 编码的图片数据
}

type UsageMetadata struct {
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	PromptTokenCount     int `json:"promptTokenCount"`
	ThoughtsTokenCount   int `json:"thoughtsTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// GeminiClient 客户端
type GeminiClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewGeminiClient(baseURL, apiKey string) *GeminiClient {
	return &GeminiClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// GenerateImage 生成图片
func (c *GeminiClient) GenerateImage(model, prompt string) (*GeminiResponse, error) {
	req := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", c.BaseURL, model, c.APIKey)

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求失败 (%d): %s", resp.StatusCode, string(body))
	}

	var result GeminiResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &result, nil
}

// SaveBase64Image 保存 Base64 编码的图片
func SaveBase64Image(base64Data, filename string) error {
	// 解码 Base64 数据
	imageData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("解码 Base64 失败: %w", err)
	}

	// 保存到文件
	if err := os.WriteFile(filename, imageData, 0644); err != nil {
		return fmt.Errorf("保存文件失败: %w", err)
	}

	return nil
}

func main() {
	client := NewGeminiClient(AntigravityBaseURL, APIKey)

	// 生成图片
	prompt := "a beautiful sunset over mountains, highly detailed, photorealistic"
	model := "gemini-3-pro-image"

	fmt.Printf("正在生成图片...\n")
	fmt.Printf("提示词: %s\n", prompt)
	fmt.Printf("模型: %s\n\n", model)

	result, err := client.GenerateImage(model, prompt)
	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		return
	}

	// 显示使用统计
	fmt.Printf("✓ 生成成功!\n")
	fmt.Printf("模型版本: %s\n", result.ModelVersion)
	fmt.Printf("响应ID: %s\n", result.ResponseID)
	fmt.Printf("Token使用情况:\n")
	fmt.Printf("  - Prompt tokens: %d\n", result.UsageMetadata.PromptTokenCount)
	fmt.Printf("  - Candidates tokens: %d\n", result.UsageMetadata.CandidatesTokenCount)
	fmt.Printf("  - Thoughts tokens: %d\n", result.UsageMetadata.ThoughtsTokenCount)
	fmt.Printf("  - Total tokens: %d\n\n", result.UsageMetadata.TotalTokenCount)

	// 提取并保存图片
	if len(result.Candidates) == 0 {
		fmt.Println("没有生成任何内容")
		return
	}

	candidate := result.Candidates[0]
	fmt.Printf("完成原因: %s\n", candidate.FinishReason)

	for i, part := range candidate.Content.Parts {
		if part.InlineData != nil && part.InlineData.Data != "" {
			filename := fmt.Sprintf("generated_image_%d.jpg", i+1)
			fmt.Printf("\n正在保存图片 %d 到 %s ...\n", i+1, filename)

			if err := SaveBase64Image(part.InlineData.Data, filename); err != nil {
				fmt.Printf("保存失败: %v\n", err)
				continue
			}

			// 获取文件大小
			fileInfo, _ := os.Stat(filename)
			fmt.Printf("✓ 保存成功! 文件大小: %.2f KB\n", float64(fileInfo.Size())/1024)
		}

		if part.Text != "" {
			fmt.Printf("\n文本内容: %s\n", part.Text)
		}
	}
}
