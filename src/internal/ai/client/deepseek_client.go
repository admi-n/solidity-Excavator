package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/admi-n/solidity-Excavator/src/internal"
)

// DeepSeekClient 实现 DeepSeek API 调用
type DeepSeekClient struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
}

// DeepSeekConfig 配置结构
type DeepSeekConfig struct {
	APIKey  string
	BaseURL string // 默认 "https://api.deepseek.com/v1"
	Model   string // 默认 "deepseek-chat"
	Timeout time.Duration
	Proxy   string // HTTP 代理
}

// DeepSeek API 请求/响应结构（与 OpenAI 兼容）
type deepSeekRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

type deepSeekResponse struct {
	ID      string    `json:"id"`
	Object  string    `json:"object"`
	Created int64     `json:"created"`
	Model   string    `json:"model"`
	Choices []Choice  `json:"choices"`
	Usage   Usage     `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

// NewDeepSeekClient 创建新的 DeepSeek 客户端
func NewDeepSeekClient(cfg DeepSeekConfig) (*DeepSeekClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com/v1"
	}

	if cfg.Model == "" {
		cfg.Model = "deepseek-chat"
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	// 配置 HTTP 客户端
	httpClient, err := internal.CreateProxyHTTPClient(cfg.Proxy, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP客户端失败: %w", err)
	}

	if cfg.Proxy != "" {
		fmt.Printf("使用代理: %s\n", cfg.Proxy)
	}

	return &DeepSeekClient{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		httpClient: httpClient,
		timeout:    cfg.Timeout,
	}, nil
}

// SendPrompt 发送 prompt 到 DeepSeek API 并返回响应
func (c *DeepSeekClient) SendPrompt(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	// 构建请求
	reqBody := deepSeekRequest{
		Model: c.model,
		Messages: []Message{
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
	var apiResp deepSeekResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查错误
	if apiResp.Error != nil {
		return "", fmt.Errorf("DeepSeek API error: %s (type: %s, code: %s)",
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
func (c *DeepSeekClient) Analyze(ctx context.Context, prompt string) (string, error) {
	// 为漏洞扫描设置系统 prompt
	systemPrompt := `You are an expert smart contract security auditor specialized in finding vulnerabilities in Solidity code.
Analyze the provided contract code carefully and identify potential security issues.
Return your analysis in a structured JSON format with clear vulnerability descriptions, severity levels, and recommendations.`

	return c.SendPrompt(ctx, systemPrompt, prompt)
}

// GetName 返回客户端名称
func (c *DeepSeekClient) GetName() string {
	return fmt.Sprintf("DeepSeek (%s)", c.model)
}

// Close 清理资源
func (c *DeepSeekClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
