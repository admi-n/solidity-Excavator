package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/admi-n/solidity-Excavator/src/internal/ai/client"
)

// AIClient 定义所有 AI 客户端必须实现的接口
type AIClient interface {
	Analyze(ctx context.Context, prompt string) (string, error)
	GetName() string
	Close() error
}

// AIClientConfig 客户端配置
type AIClientConfig struct {
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
	Timeout  time.Duration
	Proxy    string
}

// NewAIClient 根据 provider 创建对应的 AI 客户端
func NewAIClient(cfg AIClientConfig) (AIClient, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}

	switch cfg.Provider {
	case "chatgpt5", "openai", "gpt4":
		return client.NewChatGPT5Client(client.ChatGPT5Config{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
			Timeout: cfg.Timeout,
			Proxy:   cfg.Proxy,
		})

	case "deepseek":
		return client.NewDeepSeekClient(client.DeepSeekConfig{
			APIKey:  cfg.APIKey,
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
			Timeout: cfg.Timeout,
			Proxy:   cfg.Proxy,
		})

	case "local-llm", "ollama":
		return client.NewLocalLLMClient(client.LocalLLMConfig{
			BaseURL: cfg.BaseURL,
			Model:   cfg.Model,
			Timeout: cfg.Timeout,
		})

	default:
		return nil, fmt.Errorf("unsupported AI provider: %s (supported: chatgpt5, deepseek, local-llm)", cfg.Provider)
	}
}

// ValidateProvider 验证提供商名称是否有效
func ValidateProvider(provider string) error {
	validProviders := map[string]bool{
		"chatgpt5":  true,
		"openai":    true,
		"gpt4":      true,
		"deepseek":  true,
		"local-llm": true,
		"ollama":    true,
	}

	if !validProviders[provider] {
		return fmt.Errorf("invalid provider '%s', must be one of: chatgpt5, openai, gpt4, deepseek, local-llm, ollama", provider)
	}

	return nil
}
