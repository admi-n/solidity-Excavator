package parser // ✅ 修正 package 名

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
	jsonRegex := regexp.MustCompile("```(?:json)?\n?([\\s\\S]*?)\n?```")
	return &Parser{
		jsonExtractor: jsonRegex,
	}
}

// Parse 解析 AI 响应文本
func (p *Parser) Parse(response string) (*AnalysisResult, error) {
	var result AnalysisResult
	err := json.Unmarshal([]byte(response), &result)
	if err == nil {
		return &result, nil
	}

	matches := p.jsonExtractor.FindStringSubmatch(response)
	if len(matches) > 1 {
		jsonStr := strings.TrimSpace(matches[1])
		err = json.Unmarshal([]byte(jsonStr), &result)
		if err == nil {
			return &result, nil
		}
	}

	cleaned := p.cleanResponse(response)
	err = json.Unmarshal([]byte(cleaned), &result)
	if err == nil {
		return &result, nil
	}

	return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
}

func (p *Parser) cleanResponse(response string) string {
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start >= 0 && end > start {
		response = response[start : end+1]
	}

	return response
}

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

// AnalysisResult AI 分析结果结构
type AnalysisResult struct {
	ContractAddress  string          `json:"contract_address,omitempty"`
	Vulnerabilities  []Vulnerability `json:"vulnerabilities"`
	Summary          string          `json:"summary,omitempty"`
	RiskScore        float64         `json:"risk_score,omitempty"`
	Recommendations  []string        `json:"recommendations,omitempty"`
	RawResponse      string          `json:"-"`
	ParseError       string          `json:"parse_error,omitempty"`
	AnalysisDuration time.Duration   `json:"-"`
}

// Vulnerability 漏洞结构
type Vulnerability struct {
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
	Location    string   `json:"location,omitempty"`
	LineNumbers []int    `json:"line_numbers,omitempty"`
	CodeSnippet string   `json:"code_snippet,omitempty"`
	Impact      string   `json:"impact,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
	References  []string `json:"references,omitempty"`
	SWCID       string   `json:"swc_id,omitempty"`
}

func (r *AnalysisResult) GetHighSeverityCount() int {
	count := 0
	for _, v := range r.Vulnerabilities {
		if v.Severity == "Critical" || v.Severity == "High" {
			count++
		}
	}
	return count
}

func (r *AnalysisResult) HasCriticalVulnerabilities() bool {
	for _, v := range r.Vulnerabilities {
		if v.Severity == "Critical" {
			return true
		}
	}
	return false
}

func (r *AnalysisResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
