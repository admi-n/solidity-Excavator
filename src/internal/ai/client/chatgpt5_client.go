package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ChatGPT5Client 实现 OpenAI API 调用
type ChatGPT5Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// ChatGPT5Config 配置结构
type ChatGPT5Config struct {
	APIKey  string
	BaseURL string // 默认 "https://api.openai.com/v1"
	Model   string // 默认 "gpt-4" 或 "gpt-4-turbo"
	Timeout time.Duration
	Proxy   string // HTTP 代理
}

// OpenAI API 请求/响应结构
type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []choice  `json:"choices"`
	Usage   usage     `json:"usage"`
	Error   *apiError `json:"error,omitempty"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// NewChatGPT5Client 创建新的 ChatGPT-5 客户端
func NewChatGPT5Client(cfg ChatGPT5Config) (*ChatGPT5Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}

	if cfg.Model == "" {
		cfg.Model = "gpt-4-turbo" // 默认使用 GPT-4 Turbo
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	// 配置 HTTP 客户端
	transport := &http.Transport{}
	if cfg.Proxy != "" {
		// 如果需要代理支持，可以在这里配置
		fmt.Printf("使用代理: %s\n", cfg.Proxy)
	}

	httpClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	return &ChatGPT5Client{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		httpClient: httpClient,
		timeout:    cfg.Timeout,
	}, nil
}

// SendPrompt 发送 prompt 到 ChatGPT API 并返回响应
func (c *ChatGPT5Client) SendPrompt(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// 构建请求
	reqBody := openAIRequest{
		Model: c.model,
		Messages: []message{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
		Temperature: 0.1, // 较低的温度以获得更确定的结果
		MaxTokens:   4096,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// 解析响应
	var apiResp openAIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查错误
	if apiResp.Error != nil {
		return "", fmt.Errorf("OpenAI API error: %s (type: %s, code: %s)",
			apiResp.Error.Message, apiResp.Error.Type, apiResp.Error.Code)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// 提取回复内容
	if len(apiResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	content := apiResp.Choices[0].Message.Content

	// 打印 token 使用情况（可选）
	fmt.Printf("📊 Token 使用: Prompt=%d, Completion=%d, Total=%d\n",
		apiResp.Usage.PromptTokens,
		apiResp.Usage.CompletionTokens,
		apiResp.Usage.TotalTokens)

	return content, nil
}

// Analyze 分析合约代码（实现 AIClient 接口）
func (c *ChatGPT5Client) Analyze(ctx context.Context, prompt string) (string, error) {
	// 为漏洞扫描设置系统 prompt
	systemPrompt := `You are an expert smart contract security auditor specialized in finding vulnerabilities in Solidity code.
Analyze the provided contract code carefully and identify potential security issues.
Return your analysis in a structured JSON format with clear vulnerability descriptions, severity levels, and recommendations.`

	return c.SendPrompt(ctx, systemPrompt, prompt)
}

// GetName 返回客户端名称
func (c *ChatGPT5Client) GetName() string {
	return fmt.Sprintf("ChatGPT-5 (%s)", c.model)
}

// Close 清理资源
func (c *ChatGPT5Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
