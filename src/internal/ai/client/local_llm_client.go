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

// LocalLLMClient 本地 LLM 客户端（例如 Ollama）
type LocalLLMClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

// LocalLLMConfig 本地 LLM 配置
type LocalLLMConfig struct {
	BaseURL string // 例如 "http://localhost:11434"
	Model   string // 例如 "llama2", "codellama"
	Timeout time.Duration
}

// Ollama API 请求/响应结构
type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Model    string `json:"model"`
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// NewLocalLLMClient 创建本地 LLM 客户端
func NewLocalLLMClient(cfg LocalLLMConfig) (*LocalLLMClient, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:11434"
	}

	if cfg.Model == "" {
		cfg.Model = "llama2"
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 120 * time.Second // 本地模型可能需要更长时间
	}

	return &LocalLLMClient{
		baseURL: cfg.BaseURL,
		model:   cfg.Model,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}, nil
}

// Analyze 分析合约代码（实现 AIClient 接口）
func (c *LocalLLMClient) Analyze(ctx context.Context, prompt string) (string, error) {
	// 构建请求
	reqBody := ollamaRequest{
		Model:  c.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	url := fmt.Sprintf("%s/api/generate", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
	var apiResp ollamaResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 检查错误
	if apiResp.Error != "" {
		return "", fmt.Errorf("Ollama API error: %s", apiResp.Error)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return apiResp.Response, nil
}

// GetName 返回客户端名称
func (c *LocalLLMClient) GetName() string {
	return fmt.Sprintf("Local LLM (%s)", c.model)
}

// Close 清理资源
func (c *LocalLLMClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
