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
	// 首先尝试解析JSON格式
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

	// 如果JSON解析失败，尝试解析文本格式（hourglassvul模板格式）
	return p.parseTextFormat(response)
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

// parseTextFormat 解析文本格式的AI响应（hourglassvul模板格式）
func (p *Parser) parseTextFormat(response string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		RawResponse: response,
	}

	// 解析合约功能相似度
	funcSimilarity := p.extractValue(response, "合约功能相似度：", "漏洞相似度：")
	if funcSimilarity == "" {
		// 尝试其他可能的格式
		funcSimilarity = p.extractValue(response, "合约功能相识度：", "漏洞相识度：")
	}
	if funcSimilarity != "" {
		// 提取百分比数值
		funcPercent := p.extractPercentage(funcSimilarity)
		if funcPercent != "" {
			result.Summary += fmt.Sprintf("合约功能相似度: %s (%s)\n", funcPercent, funcSimilarity)
		} else {
			result.Summary += fmt.Sprintf("合约功能相似度: %s\n", funcSimilarity)
		}
	}

	// 解析漏洞相似度
	vulnSimilarity := p.extractValue(response, "漏洞相似度：", "可能存在类似漏洞概率：")
	if vulnSimilarity == "" {
		// 尝试其他可能的格式
		vulnSimilarity = p.extractValue(response, "漏洞相识度：", "可能存在漏洞概率")
	}
	if vulnSimilarity != "" {
		// 提取百分比数值
		vulnPercent := p.extractPercentage(vulnSimilarity)
		if vulnPercent != "" {
			result.Summary += fmt.Sprintf("漏洞相似度: %s (%s)\n", vulnPercent, vulnSimilarity)
		} else {
			result.Summary += fmt.Sprintf("漏洞相似度: %s\n", vulnSimilarity)
		}
	}

	// 解析漏洞概率
	probability := p.extractValue(response, "可能存在类似漏洞概率：", "漏洞等级：")
	if probability == "" {
		// 尝试其他可能的格式
		probability = p.extractValue(response, "可能存在漏洞概率", "漏洞等级")
	}
	if probability != "" {
		result.Summary += fmt.Sprintf("漏洞概率: %s\n", probability)
	}

	// 解析漏洞等级
	severity := p.extractValue(response, "漏洞等级:", "")
	if severity == "" {
		// 尝试其他可能的格式
		severity = p.extractValue(response, "漏洞等级：", "")
	}

	// 添加调试信息
	result.Summary += fmt.Sprintf("解析的漏洞等级: '%s'\n", severity)

	if severity != "" {
		// 根据漏洞等级创建漏洞记录
		// 只有当漏洞等级不是"无"、"无漏洞"、"未发现漏洞"、"低"时才创建漏洞记录
		// 对于"中"、"高"、"严重"等级，都认为是存在漏洞
		if severity != "无" && severity != "无漏洞" && severity != "未发现漏洞" && severity != "低" {
			vuln := Vulnerability{
				Type:        "Hourglass Multi-Level Referral Commission",
				Severity:    p.mapSeverity(severity),
				Description: fmt.Sprintf("基于复现代码分析，合约功能相似度: %s, 漏洞相似度: %s, 漏洞概率: %s", funcSimilarity, vulnSimilarity, probability),
			}
			result.Vulnerabilities = append(result.Vulnerabilities, vuln)
			result.Summary += fmt.Sprintf("已创建漏洞记录，严重等级: %s\n", p.mapSeverity(severity))
		} else {
			result.Summary += fmt.Sprintf("漏洞等级为'%s'，不创建漏洞记录\n", severity)
		}
	} else {
		result.Summary += "未找到漏洞等级信息\n"
	}

	return result, nil
}

// extractValue 从文本中提取指定键的值
func (p *Parser) extractValue(text, startKey, endKey string) string {
	startIndex := strings.Index(text, startKey)
	if startIndex == -1 {
		return ""
	}

	startIndex += len(startKey)
	var endIndex int

	if endKey != "" {
		endIndex = strings.Index(text[startIndex:], endKey)
		if endIndex == -1 {
			endIndex = len(text)
		} else {
			endIndex += startIndex
		}
	} else {
		// 如果没有结束键，找到下一个换行符
		endIndex = strings.Index(text[startIndex:], "\n")
		if endIndex == -1 {
			endIndex = len(text)
		} else {
			endIndex += startIndex
		}
	}

	value := strings.TrimSpace(text[startIndex:endIndex])
	return value
}

// mapSeverity 将中文严重等级映射到标准等级
func (p *Parser) mapSeverity(severity string) string {
	severity = strings.TrimSpace(severity)
	switch severity {
	case "低":
		return "Low"
	case "中":
		return "Medium"
	case "高":
		return "High"
	case "严重":
		return "Critical"
	default:
		return "Unknown"
	}
}

// extractPercentage 从文本中提取百分比数值
func (p *Parser) extractPercentage(text string) string {
	// 使用正则表达式匹配百分比
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)%`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1] + "%"
	}
	return ""
}
