package client

// 共享的 API 类型定义

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Choice 选择结构
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 使用情况结构
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError API 错误结构
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}
