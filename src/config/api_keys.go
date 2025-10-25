package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AIConfig AI 相关配置
type AIConfig struct {
	OpenAI struct {
		APIKey  string `yaml:"api_key"`
		BaseURL string `yaml:"base_url"` // 可选，默认使用官方 API
		Model   string `yaml:"model"`    // 可选，默认 gpt-4-turbo
	} `yaml:"openai"`

	DeepSeek struct {
		APIKey  string `yaml:"api_key"`
		BaseURL string `yaml:"base_url"` // 默认 https://api.deepseek.com/v1
		Model   string `yaml:"model"`    // 默认 deepseek-chat
	} `yaml:"deepseek"`

	LocalLLM struct {
		BaseURL string `yaml:"base_url"` // 例如 http://localhost:11434
		Model   string `yaml:"model"`    // 例如 llama2
	} `yaml:"local_llm"`
}

// Settings 全局配置结构（扩展现有的配置）
type Settings struct {
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`

	RPC struct {
		Ethereum string `yaml:"ethereum"`
		BSC      string `yaml:"bsc"`
		Arbitrum string `yaml:"arbitrum"`
	} `yaml:"rpc"`

	AI AIConfig `yaml:"ai"`
}

var globalSettings *Settings

// LoadSettings 加载配置文件
func LoadSettings(configPath string) error {
	if configPath == "" {
		configPath = "config/settings.yaml"
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var settings Settings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	globalSettings = &settings
	return nil
}

// GetOpenAIKey 获取 OpenAI API Key
func GetOpenAIKey() (string, error) {
	// 优先从环境变量读取
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key, nil
	}

	// 从配置文件读取
	if globalSettings == nil {
		if err := LoadSettings(""); err != nil {
			return "", err
		}
	}

	if globalSettings.AI.OpenAI.APIKey == "" {
		return "", fmt.Errorf("OpenAI API key not found in config or environment variable OPENAI_API_KEY")
	}

	return globalSettings.AI.OpenAI.APIKey, nil
}

// GetOpenAIBaseURL 获取 OpenAI Base URL
func GetOpenAIBaseURL() string {
	if globalSettings == nil {
		LoadSettings("")
	}

	if globalSettings != nil && globalSettings.AI.OpenAI.BaseURL != "" {
		return globalSettings.AI.OpenAI.BaseURL
	}

	return "https://api.openai.com/v1" // 默认值
}

// GetOpenAIModel 获取 OpenAI 模型名称
func GetOpenAIModel() string {
	if globalSettings == nil {
		LoadSettings("")
	}

	if globalSettings != nil && globalSettings.AI.OpenAI.Model != "" {
		return globalSettings.AI.OpenAI.Model
	}

	return "gpt-4-turbo" // 默认值
}

// GetDeepSeekKey 获取 DeepSeek API Key
func GetDeepSeekKey() (string, error) {
	// 优先从环境变量读取
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		return key, nil
	}

	// 从配置文件读取
	if globalSettings == nil {
		if err := LoadSettings(""); err != nil {
			return "", err
		}
	}

	if globalSettings.AI.DeepSeek.APIKey == "" {
		return "", fmt.Errorf("DeepSeek API key not found in config or environment variable DEEPSEEK_API_KEY")
	}

	return globalSettings.AI.DeepSeek.APIKey, nil
}

// GetDeepSeekBaseURL 获取 DeepSeek Base URL
func GetDeepSeekBaseURL() string {
	if globalSettings == nil {
		LoadSettings("")
	}

	if globalSettings != nil && globalSettings.AI.DeepSeek.BaseURL != "" {
		return globalSettings.AI.DeepSeek.BaseURL
	}

	return "https://api.deepseek.com/v1" // 默认值
}

// GetDeepSeekModel 获取 DeepSeek 模型名称
func GetDeepSeekModel() string {
	if globalSettings == nil {
		LoadSettings("")
	}

	if globalSettings != nil && globalSettings.AI.DeepSeek.Model != "" {
		return globalSettings.AI.DeepSeek.Model
	}

	return "deepseek-chat" // 默认值
}

// GetLocalLLMConfig 获取本地 LLM 配置
func GetLocalLLMConfig() (baseURL, model string) {
	if globalSettings == nil {
		LoadSettings("")
	}

	if globalSettings != nil {
		baseURL = globalSettings.AI.LocalLLM.BaseURL
		model = globalSettings.AI.LocalLLM.Model
	}

	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "llama2"
	}

	return baseURL, model
}
