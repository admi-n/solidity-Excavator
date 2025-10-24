package ai

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Parser 解析 AI 返回的分析结果
type Parser struct {
	jsonExtractor *regexp.Regexp
}

// NewParser 创建新的解析器
func NewParser() *Parser {
	// 用于提取 JSON 代码块的正则表达式
	jsonRegex := regexp.MustCompile("```(?:json)?\n?([\\s\\S]*?)\n?```")

	return &Parser{
		jsonExtractor: jsonRegex,
	}
}

// Parse 解析 AI 响应文本
func (p *Parser) Parse(response string) (*AnalysisResult, error) {
	// 尝试直接解析 JSON
	var result AnalysisResult
	err := json.Unmarshal([]byte(response), &result)
	if err == nil {
		return &result, nil
	}

	// 尝试从 markdown 代码块中提取 JSON
	matches := p.jsonExtractor.FindStringSubmatch(response)
	if len(matches) > 1 {
		jsonStr := strings.TrimSpace(matches[1])
		err = json.Unmarshal([]byte(jsonStr), &result)
		if err == nil {
			return &result, nil
		}
	}

	// 如果仍然失败，尝试清理响应并再次解析
	cleaned := p.cleanResponse(response)
	err = json.Unmarshal([]byte(cleaned), &result)
	if err == nil {
		return &result, nil
	}

	// 解析失败，返回错误
	return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
}

// cleanResponse 清理响应文本
func (p *Parser) cleanResponse(response string) string {
	// 移除常见的非 JSON 前缀
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// 尝试找到第一个 { 和最后一个 }
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	return response
}

// ParseVulnerabilities 从文本中提取漏洞信息（备用方法）
func (p *Parser) ParseVulnerabilities(response string) ([]Vulnerability, error) {
	// 这是一个备用解析方法，用于处理非结构化响应
	vulnerabilities := []Vulnerability{}

	// 简单的基于关键词的解析
	lines := strings.Split(response, "\n")
	var currentVuln *Vulnerability

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检测漏洞标题
		if strings.Contains(strings.ToLower(line), "vulnerability") ||
			strings.Contains(strings.ToLower(line), "issue") ||
			strings.HasPrefix(line, "##") {
			if currentVuln != nil {
				vulnerabilities = append(vulnerabilities, *currentVuln)
			}
			currentVuln = &Vulnerability{
				Type:        extractVulnType(line),
				Description: line,
			}
		} else if currentVuln != nil {
			// 累积描述
			currentVuln.Description += " " + line
		}
	}

	if currentVuln != nil {
		vulnerabilities = append(vulnerabilities, *currentVuln)
	}

	return vulnerabilities, nil
}

// extractVulnType 从文本中提取漏洞类型
func extractVulnType(text string) string {
	text = strings.ToLower(text)

	vulnTypes := map[string]string{
		"reentrancy":           "Reentrancy",
		"integer overflow":     "Integer Overflow",
		"integer underflow":    "Integer Underflow",
		"unchecked call":       "Unchecked External Call",
		"access control":       "Access Control",
		"denial of service":    "Denial of Service",
		"timestamp dependence": "Timestamp Dependence",
		"front running":        "Front Running",
		"delegatecall":         "Delegatecall",
		"selfdestruct":         "Selfdestruct",
	}

	for keyword, vulnType := range vulnTypes {
		if strings.Contains(text, keyword) {
			return vulnType
		}
	}

	return "Unknown"
}

// ValidateResult 验证解析结果的有效性
func (p *Parser) ValidateResult(result *AnalysisResult) error {
	if result == nil {
		return fmt.Errorf("analysis result is nil")
	}

	// 验证每个漏洞
	for i, vuln := range result.Vulnerabilities {
		if vuln.Type == "" {
			return fmt.Errorf("vulnerability %d missing type", i)
		}
		if vuln.Description == "" {
			return fmt.Errorf("vulnerability %d missing description", i)
		}
		if vuln.Severity == "" {
			vuln.Severity = "Medium" // 默认中等严重性
		}
	}

	return nil
}

// AnalysisResult AI 分析结果结构
type AnalysisResult struct {
	ContractAddress  string          `json:"contract_address,omitempty"`
	Vulnerabilities  []Vulnerability `json:"vulnerabilities"`
	Summary          string          `json:"summary,omitempty"`
	RiskScore        float64         `json:"risk_score,omitempty"`
	Recommendations  []string        `json:"recommendations,omitempty"`
	RawResponse      string          `json:"-"` // 原始响应，不序列化
	ParseError       string          `json:"parse_error,omitempty"`
	AnalysisDuration time.Duration   `json:"-"`
}

// Vulnerability 漏洞结构
type Vulnerability struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"` // Critical, High, Medium, Low
	Description string   `json:"description"`
	Location    string   `json:"location,omitempty"`     // 代码位置
	LineNumbers []int    `json:"line_numbers,omitempty"` // 行号
	CodeSnippet string   `json:"code_snippet,omitempty"` // 代码片段
	Impact      string   `json:"impact,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	References  []string `json:"references,omitempty"`
	SWCID       string   `json:"swc_id,omitempty"` // SWC 编号
}

// GetHighSeverityCount 获取高危漏洞数量
func (r *AnalysisResult) GetHighSeverityCount() int {
	count := 0
	for _, v := range r.Vulnerabilities {
		if v.Severity == "Critical" || v.Severity == "High" {
			count++
		}
	}
	return count
}

// HasCriticalVulnerabilities 是否存在严重漏洞
func (r *AnalysisResult) HasCriticalVulnerabilities() bool {
	for _, v := range r.Vulnerabilities {
		if v.Severity == "Critical" {
			return true
		}
	}
	return false
}

// ToJSON 转换为 JSON 字符串
func (r *AnalysisResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
