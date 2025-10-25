package internal

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ProxyConfig 代理配置
type ProxyConfig struct {
	URL     string        // 代理URL，例如 http://127.0.0.1:7897
	Timeout time.Duration // 超时时间
}

// ProxyManager 代理管理器
type ProxyManager struct {
	config *ProxyConfig
}

// NewProxyManager 创建代理管理器
func NewProxyManager(proxyURL string, timeout time.Duration) (*ProxyManager, error) {
	if strings.TrimSpace(proxyURL) == "" {
		return &ProxyManager{config: nil}, nil
	}

	// 验证代理URL格式
	_, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	return &ProxyManager{
		config: &ProxyConfig{
			URL:     strings.TrimSpace(proxyURL),
			Timeout: timeout,
		},
	}, nil
}

// CreateHTTPClient 创建带代理的HTTP客户端
func (pm *ProxyManager) CreateHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout: timeout,
	}

	if pm.config != nil {
		proxyURL, _ := url.Parse(pm.config.URL)
		client.Transport = &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     30 * time.Second,
		}
	}

	return client
}

// CreateHTTPTransport 创建带代理的HTTP Transport
func (pm *ProxyManager) CreateHTTPTransport() *http.Transport {
	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		IdleConnTimeout:     30 * time.Second,
	}

	if pm.config != nil {
		proxyURL, _ := url.Parse(pm.config.URL)
		transport.Proxy = http.ProxyURL(proxyURL)
	}

	return transport
}

// SetGlobalTransport 设置全局HTTP Transport
func (pm *ProxyManager) SetGlobalTransport() {
	if pm.config != nil {
		proxyURL, _ := url.Parse(pm.config.URL)
		http.DefaultTransport = &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     30 * time.Second,
		}
	}
}

// IsEnabled 检查代理是否启用
func (pm *ProxyManager) IsEnabled() bool {
	return pm.config != nil
}

// GetProxyURL 获取代理URL
func (pm *ProxyManager) GetProxyURL() string {
	if pm.config != nil {
		return pm.config.URL
	}
	return ""
}

// ValidateProxyURL 验证代理URL格式
func ValidateProxyURL(proxyURL string) error {
	if strings.TrimSpace(proxyURL) == "" {
		return nil // 空字符串表示不使用代理
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return fmt.Errorf("invalid proxy URL format: %w", err)
	}

	// 检查协议
	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5" {
		return fmt.Errorf("unsupported proxy scheme: %s (supported: http, https, socks5)", u.Scheme)
	}

	// 检查主机名
	if u.Host == "" {
		return fmt.Errorf("proxy host cannot be empty")
	}

	return nil
}

// CreateProxyHTTPClient 便捷函数：创建带代理的HTTP客户端
func CreateProxyHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	pm, err := NewProxyManager(proxyURL, timeout)
	if err != nil {
		return nil, err
	}

	return pm.CreateHTTPClient(timeout), nil
}

// CreateProxyTransport 便捷函数：创建带代理的HTTP Transport
func CreateProxyTransport(proxyURL string) (*http.Transport, error) {
	pm, err := NewProxyManager(proxyURL, 0)
	if err != nil {
		return nil, err
	}

	return pm.CreateHTTPTransport(), nil
}

// SetGlobalProxy 便捷函数：设置全局代理
func SetGlobalProxy(proxyURL string) error {
	pm, err := NewProxyManager(proxyURL, 0)
	if err != nil {
		return err
	}

	pm.SetGlobalTransport()
	return nil
}
