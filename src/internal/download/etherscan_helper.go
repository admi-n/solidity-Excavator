package download

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// EtherscanConfig Etherscan API 配置
type EtherscanConfig struct {
	APIKey  string
	BaseURL string
	Proxy   string // 新增：可选的 HTTP 代理 URL（例如 http://127.0.0.1:7897）
}

// EtherscanResponse Etherscan API 响应结构
type EtherscanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  []struct {
		SourceCode           string `json:"SourceCode"`
		ABI                  string `json:"ABI"`
		ContractName         string `json:"ContractName"`
		CompilerVersion      string `json:"CompilerVersion"`
		OptimizationUsed     string `json:"OptimizationUsed"`
		Runs                 string `json:"Runs"`
		ConstructorArguments string `json:"ConstructorArguments"`
		EVMVersion           string `json:"EVMVersion"`
		Library              string `json:"Library"`
		LicenseType          string `json:"LicenseType"`
		Proxy                string `json:"Proxy"`
		Implementation       string `json:"Implementation"`
		SwarmSource          string `json:"SwarmSource"`
	} `json:"result"`
}

// GetContractSource 从 Etherscan 获取合约源代码和验证状态
func GetContractSource(address string, config EtherscanConfig) (sourceCode string, isVerified bool, err error) {
	// 清理输入
	address = strings.TrimSpace(address)
	if address == "" {
		return "", false, fmt.Errorf("空的地址传入 GetContractSource")
	}

	// 构建 API URL 使用 url.Values 避免拼接错误
	base := strings.TrimRight(config.BaseURL, "/")
	u, err := url.Parse(base)
	if err != nil {
		return "", false, fmt.Errorf("解析 Etherscan BaseURL 失败: %w", err)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api"

	q := url.Values{}
	q.Set("module", "contract")
	q.Set("action", "getsourcecode")
	q.Set("address", address)
	q.Set("apikey", strings.TrimSpace(config.APIKey))
	// chainid 可选保留
	q.Set("chainid", "1")

	u.RawQuery = q.Encode()
	finalURL := u.String()

	// 准备 HTTP 客户端（超时与可选代理）
	client := &http.Client{
		Timeout: 20 * time.Second, // 稍微延长超时时间以减少偶发超时
	}

	if strings.TrimSpace(config.Proxy) != "" {
		if pu, perr := url.Parse(config.Proxy); perr == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(pu),
			}
		} else {
			// 代理解析失败直接返回错误，调用方会回退为字节码
			return "", false, fmt.Errorf("解析 Etherscan proxy 失败: %w", perr)
		}
	}

	// 重试逻辑：短暂网络错误/EOF/超时时重试
	var lastErr error
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 创建请求并加上 User-Agent
		req, _ := http.NewRequest("GET", finalURL, nil)
		req.Header.Set("User-Agent", "solidity-excavator/1.0 (+https://github.com/)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			// 判断是否为临时或超时错误，如果是则重试
			if isTemporaryNetErr(err) && attempt < maxAttempts {
				sleep := time.Duration(attempt) * 500 * time.Millisecond
				time.Sleep(sleep)
				continue
			}
			// 非临时错误或最后一次尝试 -> 返回网络错误
			return "", false, fmt.Errorf("请求 Etherscan API 失败: %w (url=%s)", err, finalURL)
		}

		// 确保关闭响应体
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			// 对于意外 EOF 等可重试的读取错误做重试
			if (readErr == io.ErrUnexpectedEOF || isTemporaryNetErr(readErr)) && attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
				continue
			}
			return "", false, fmt.Errorf("读取 Etherscan 响应失败: %w (url=%s)", readErr, finalURL)
		}

		// 检查 HTTP 状态码
		if resp.StatusCode != http.StatusOK {
			// 返回 body 片段有助于定位错误
			snippet := string(body)
			if len(snippet) > 1024 {
				snippet = snippet[:1024]
			}
			return "", false, fmt.Errorf("Etherscan 返回非 200 状态: %d, body: %s", resp.StatusCode, snippet)
		}

		// 解析 JSON
		var etherscanResp EtherscanResponse
		if jerr := json.Unmarshal(body, &etherscanResp); jerr != nil {
			lastErr = jerr
			// JSON 解析错误通常不可恢复，但做少量重试以应对偶发损坏
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt) * 300 * time.Millisecond)
				continue
			}
			return "", false, fmt.Errorf("解析 Etherscan JSON 失败: %w (url=%s)", jerr, finalURL)
		}

		// 如果 API 返回 status != "1"，则表示未验证或其它业务层面的问题（不是网络错误）
		if etherscanResp.Status != "1" {
			return "", false, nil
		}

		// 找到结果并检查 SourceCode
		if len(etherscanResp.Result) == 0 {
			return "", false, nil
		}
		res := etherscanResp.Result[0]
		if strings.TrimSpace(res.SourceCode) == "" {
			// 合约未验证
			return "", false, nil
		}
		// 成功获取已验证源码
		return res.SourceCode, true, nil
	}

	// 所有尝试失败，返回最后一个错误
	if lastErr != nil {
		return "", false, fmt.Errorf("请求 Etherscan 多次失败: %w (url=%s)", lastErr, finalURL)
	}
	return "", false, fmt.Errorf("请求 Etherscan 未知错误 (url=%s)", finalURL)
}

// isTemporaryNetErr 判断是否为可重试的网络错误
func isTemporaryNetErr(err error) bool {
	if err == nil {
		return false
	}
	// net.Error 暴露 Timeout() / Temporary()
	if ne, ok := err.(net.Error); ok {
		return ne.Timeout() || ne.Temporary()
	}
	// 常见的 IO 错误也视为临时
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		return true
	}
	// 其余默认不重试
	return false
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	ticker *time.Ticker
}

// NewRateLimiter 创建速率限制器（每秒最多 requestsPerSecond 个请求）
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	interval := time.Second / time.Duration(requestsPerSecond)
	return &RateLimiter{
		ticker: time.NewTicker(interval),
	}
}

// Wait 等待直到可以发送下一个请求
func (r *RateLimiter) Wait() {
	<-r.ticker.C
}

// Stop 停止速率限制器
func (r *RateLimiter) Stop() {
	r.ticker.Stop()
}
