package ai

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/admi-n/solidity-Excavator/src/config"
	"github.com/admi-n/solidity-Excavator/src/internal/ai/parser"
)

// Manager 管理 AI 客户端和分析请求
type Manager struct {
	client    AIClient
	parser    *parser.Parser
	rateLimit *rateLimiter
	mu        sync.Mutex
}

type rateLimiter struct {
	requests chan struct{}
	interval time.Duration
}

func newRateLimiter(requestsPerMinute int) *rateLimiter {
	rl := &rateLimiter{
		requests: make(chan struct{}, requestsPerMinute),
		interval: time.Minute / time.Duration(requestsPerMinute),
	}

	for i := 0; i < requestsPerMinute; i++ {
		rl.requests <- struct{}{}
	}

	go func() {
		ticker := time.NewTicker(rl.interval)
		defer ticker.Stop()
		for range ticker.C {
			select {
			case rl.requests <- struct{}{}:
			default:
			}
		}
	}()

	return rl
}

func (rl *rateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.requests:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type ManagerConfig struct {
	Provider       string
	APIKey         string
	BaseURL        string
	Model          string
	Timeout        time.Duration
	Proxy          string
	RequestsPerMin int
}

// NewManager 创建新的 AI 管理器
func NewManager(cfg ManagerConfig) (*Manager, error) {
	// 如果没有提供 APIKey，尝试从配置文件读取
	if cfg.APIKey == "" && (cfg.Provider == "chatgpt5" || cfg.Provider == "openai" || cfg.Provider == "gpt4") {
		apiKey, err := config.GetOpenAIKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get OpenAI API key from config: %w", err)
		}
		cfg.APIKey = apiKey
	}

	// 支持 DeepSeek
	if cfg.APIKey == "" && cfg.Provider == "deepseek" {
		apiKey, err := config.GetDeepSeekKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get DeepSeek API key from config: %w", err)
		}
		cfg.APIKey = apiKey
	}

	// 创建 AI 客户端
	client, err := NewAIClient(AIClientConfig{
		Provider: cfg.Provider,
		APIKey:   cfg.APIKey,
		BaseURL:  cfg.BaseURL,
		Model:    cfg.Model,
		Timeout:  cfg.Timeout,
		Proxy:    cfg.Proxy,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AI client: %w", err)
	}

	if cfg.RequestsPerMin <= 0 {
		cfg.RequestsPerMin = 20
	}

	return &Manager{
		client:    client,
		parser:    parser.NewParser(),
		rateLimit: newRateLimiter(cfg.RequestsPerMin),
	}, nil
}

// AnalyzeContract 分析合约代码并返回结构化结果
func (m *Manager) AnalyzeContract(ctx context.Context, contractCode, prompt string) (*parser.AnalysisResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.rateLimit.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit wait failed: %w", err)
	}

	fmt.Printf("🤖 正在使用 %s 分析合约...\n", m.client.GetName())

	fullPrompt := fmt.Sprintf("%s\n\n合约代码:\n```solidity\n%s\n```", prompt, contractCode)

	startTime := time.Now()
	response, err := m.client.Analyze(ctx, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("AI analysis failed: %w", err)
	}
	duration := time.Since(startTime)

	fmt.Printf("✅ 分析完成，耗时: %v\n", duration)

	result, err := m.parser.Parse(response)
	if err != nil {
		fmt.Printf("⚠️  解析结果失败: %v，返回原始响应\n", err)
		return &parser.AnalysisResult{
			RawResponse: response,
			ParseError:  err.Error(),
		}, nil
	}

	result.RawResponse = response
	result.AnalysisDuration = duration

	return result, nil
}

// AnalyzeBatch 批量分析多个合约
func (m *Manager) AnalyzeBatch(ctx context.Context, contracts []ContractInput, concurrency int) ([]*parser.AnalysisResult, error) {
	if concurrency <= 0 {
		concurrency = 1
	}

	results := make([]*parser.AnalysisResult, len(contracts))
	errChan := make(chan error, len(contracts))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i, contract := range contracts {
		wg.Add(1)
		go func(idx int, c ContractInput) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := m.AnalyzeContract(ctx, c.Code, c.Prompt)
			if err != nil {
				errChan <- fmt.Errorf("contract %d (%s) failed: %w", idx, c.Address, err)
				return
			}

			result.ContractAddress = c.Address
			results[idx] = result
		}(i, contract)
	}

	wg.Wait()
	close(errChan)

	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("batch analysis completed with %d errors: %v", len(errs), errs[0])
	}

	return results, nil
}

type ContractInput struct {
	Address string
	Code    string
	Prompt  string
}

func (m *Manager) GetClientInfo() string {
	return m.client.GetName()
}

func (m *Manager) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}

func (m *Manager) TestConnection(ctx context.Context) error {
	fmt.Println("🔍 测试 AI 客户端连接...")

	testPrompt := "Please respond with 'OK' if you can read this message."
	_, err := m.client.Analyze(ctx, testPrompt)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	fmt.Println("✅ AI 客户端连接成功!")
	return nil
}
