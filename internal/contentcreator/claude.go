package contentcreator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/crosszan/modu/pkg/env"
)

// ClaudeClient OpenAI 兼容 API 客户端（用于本地代理）
type ClaudeClient struct {
	APIKey  string
	BaseURL string
	Model   string
	client  *http.Client
}

// NewClaudeClient 创建客户端
func NewClaudeClient() *ClaudeClient {
	// API Key: 优先使用 OPENAI_API_KEY，否则使用 LOCAL_PROXY_API_KEY
	apiKey := env.GetDefault("OPENAI_API_KEY", env.GetDefault("LOCAL_PROXY_API_KEY", "sk-5fec10a3ada64c0b808122ee2b971a5d"))

	// Base URL: 优先使用 OPENAI_BASE_URL，否则使用 LOCAL_PROXY_URL，默认 localhost
	baseURL := env.GetDefault("OPENAI_BASE_URL", env.GetDefault("LOCAL_PROXY_URL", "http://127.0.0.1:8045/v1"))

	// Model: 优先使用 OPENAI_MODEL，否则使用 LOCAL_PROXY_MODEL，默认 gpt-4
	model := env.GetDefault("OPENAI_MODEL", env.GetDefault("LOCAL_PROXY_MODEL", "gemini-3-pro-high"))

	return &ClaudeClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// ChatMessage OpenAI 兼容消息结构
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionRequest OpenAI 兼容请求
type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatCompletionResponse OpenAI 兼容响应
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Complete 调用 OpenAI 兼容 API
func (c *ClaudeClient) Complete(systemPrompt, userPrompt string, maxTokens int) (string, error) {
	if c.APIKey == "" {
		return "", fmt.Errorf("API key not set (OPENAI_API_KEY or LOCAL_PROXY_API_KEY)")
	}

	if maxTokens == 0 {
		maxTokens = 4096
	}

	messages := []ChatMessage{}
	if systemPrompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: userPrompt,
	})

	req := ChatCompletionRequest{
		Model:       c.Model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: 0.7,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatCompletionResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// AnalyzeTrends 分析热点趋势
func (c *ClaudeClient) AnalyzeTrends(topics []HotTopic, limit int) (*TrendAnalysis, error) {
	if limit <= 0 {
		limit = len(topics)
	}
	if limit > len(topics) {
		limit = len(topics)
	}

	// 构建热点列表
	topicsJSON, _ := json.MarshalIndent(topics[:limit], "", "  ")

	systemPrompt := `你是一个资深的内容运营专家，擅长分析热点话题的流量潜力。

请分析以下热点话题，识别出最具传播潜力的 5 个选题，并给出推荐理由。

评估标准：
1. 争议性（能否引发讨论）
2. 紧迫感（时效性强不强）
3. 好奇心（是否有信息落差）
4. 普适性（受众面是否够广）

输出格式使用 JSON：
{
  "recommendations": [
    {
      "topic": "话题标题",
      "reason": "推荐理由（为什么有流量潜力）",
      "potential": 85,
      "angle": "建议的切入角度",
      "source": "来源"
    }
  ],
  "summary": "总体趋势分析"
}`

	userPrompt := fmt.Sprintf("请分析以下热点话题：\n\n%s", string(topicsJSON))

	resp, err := c.Complete(systemPrompt, userPrompt, 2048)
	if err != nil {
		return nil, err
	}

	// 解析 JSON 响应
	var result struct {
		Recommendations []TopicRecommendation `json:"recommendations"`
		Summary         string                `json:"summary"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		// 如果不是 JSON，返回原始文本
		return &TrendAnalysis{
			Topics:  topics[:limit],
			Summary: resp,
		}, nil
	}

	return &TrendAnalysis{
		Topics:          topics[:limit],
		Recommendations: result.Recommendations,
		Summary:         result.Summary,
	}, nil
}

// GenerateContent 生成内容
func (c *ClaudeClient) GenerateContent(req *GenerateRequest, stylePrompt string) (string, error) {
	var userPrompt string

	if req.Topic != "" {
		userPrompt = fmt.Sprintf("请基于以下话题创作内容：\n\n话题：%s", req.Topic)
	} else if req.RawContent != "" {
		userPrompt = fmt.Sprintf("请优化以下草稿内容：\n\n%s", req.RawContent)
	} else if len(req.Sources) > 0 {
		sourcesJSON, _ := json.MarshalIndent(req.Sources, "", "  ")
		userPrompt = fmt.Sprintf("请基于以下热点话题选择最佳切入点并创作内容：\n\n%s", string(sourcesJSON))
	} else {
		return "", fmt.Errorf("no topic or content provided")
	}

	resp, err := c.Complete(stylePrompt, userPrompt, 4096)
	if err != nil {
		return "", err
	}

	return resp, nil
}

// VerifyContent 验证内容
func (c *ClaudeClient) VerifyContent(content string, verificationPrompt string) (string, error) {
	userPrompt := fmt.Sprintf("请验证以下内容的爆款潜力：\n\n%s", content)

	resp, err := c.Complete(verificationPrompt, userPrompt, 4096)
	if err != nil {
		return "", err
	}

	return resp, nil
}
