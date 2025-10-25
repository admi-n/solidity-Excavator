package report

import (
	"fmt"
	"time"
)

// ScanResult 表示单次扫描的结果
type ScanResult struct {
	ContractAddress string
	ScanTime        time.Time
	Status          string
	Vulnerabilities []Vulnerability
	AnalysisSummary string
	RawResponse     string
}

// Vulnerability 表示发现的漏洞
type Vulnerability struct {
	Type        string
	Severity    string
	Description string
}

// Report 表示完整的扫描报告
type Report struct {
	Mode                 string
	Strategy             string
	AIProvider           string
	ScanTime             time.Time
	TotalContracts       int
	VulnerableContracts  int
	SeverityDistribution map[string]int
	Results              []ScanResult
}

// Generator 报告生成器接口
type Generator interface {
	Generate(report *Report) (string, error)
}

// MarkdownGenerator markdown格式报告生成器
type MarkdownGenerator struct{}

// NewMarkdownGenerator 创建markdown报告生成器
func NewMarkdownGenerator() *MarkdownGenerator {
	return &MarkdownGenerator{}
}

// Generate 生成markdown格式报告
func (g *MarkdownGenerator) Generate(report *Report) (string, error) {
	var result string

	// 报告头部
	result += fmt.Sprintf("# Solidity Excavator 扫描报告\n\n")
	result += fmt.Sprintf("**扫描模式**: %s\n", report.Mode)
	result += fmt.Sprintf("**策略**: %s\n", report.Strategy)
	result += fmt.Sprintf("**AI 提供商**: %s\n", report.AIProvider)
	result += fmt.Sprintf("**扫描时间**: %s\n\n", report.ScanTime.Format("2006-01-02 15:04:05"))

	// 扫描统计
	result += fmt.Sprintf("## 扫描统计\n\n")
	result += fmt.Sprintf("- **总合约数**: %d\n", report.TotalContracts)
	result += fmt.Sprintf("- **存在漏洞**: %d\n\n", report.VulnerableContracts)

	// 漏洞严重性分布
	if len(report.SeverityDistribution) > 0 {
		result += fmt.Sprintf("## 漏洞严重性分布\n\n")
		for severity, count := range report.SeverityDistribution {
			result += fmt.Sprintf("- **%s**: %d\n", severity, count)
		}
		result += "\n"
	}

	// 详细结果
	result += fmt.Sprintf("## 详细结果\n\n")

	for i, scanResult := range report.Results {
		// 合约地址作为一级标题
		result += fmt.Sprintf("# 合约地址: %s\n\n", scanResult.ContractAddress)
		result += fmt.Sprintf("**扫描时间**: %s\n", scanResult.ScanTime.Format("2006-01-02 15:04:05"))
		result += fmt.Sprintf("**状态**: %s\n\n", scanResult.Status)

		// AI分析摘要
		if scanResult.AnalysisSummary != "" {
			result += fmt.Sprintf("### AI分析摘要\n\n")
			result += fmt.Sprintf("%s\n\n", scanResult.AnalysisSummary)
		}

		// 漏洞详情
		if len(scanResult.Vulnerabilities) > 0 {
			result += fmt.Sprintf("### 漏洞详情\n\n")
			for j, vuln := range scanResult.Vulnerabilities {
				severityIcon := getSeverityIcon(vuln.Severity)
				result += fmt.Sprintf("%d. %s **[%s]** %s\n", j+1, severityIcon, vuln.Severity, vuln.Type)
				result += fmt.Sprintf("   **描述**: %s\n\n", vuln.Description)
			}
		}

		// 原始AI响应（可选）
		if scanResult.RawResponse != "" {
			result += fmt.Sprintf("### AI原始响应\n\n")
			result += fmt.Sprintf("```\n%s\n```\n\n", scanResult.RawResponse)
		}

		// 如果不是最后一个结果，添加分隔线
		if i < len(report.Results)-1 {
			result += fmt.Sprintf("---\n\n")
		}
	}

	return result, nil
}

// getSeverityIcon 获取严重等级对应的图标
func getSeverityIcon(severity string) string {
	switch severity {
	case "Critical":
		return "🔴"
	case "High":
		return "🟠"
	case "Medium":
		return "🟡"
	case "Low":
		return "🟢"
	default:
		return "⚪"
	}
}
