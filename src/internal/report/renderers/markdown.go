package renderers

import (
	"fmt"
	"strings"
)

// MarkdownRenderer markdown渲染器
type MarkdownRenderer struct{}

// NewMarkdownRenderer 创建markdown渲染器
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{}
}

// RenderVulnerability 渲染单个漏洞
func (r *MarkdownRenderer) RenderVulnerability(vulnType, severity, description string) string {
	icon := getSeverityIcon(severity)
	return fmt.Sprintf("%s **[%s]** %s\n   **描述**: %s", icon, severity, vulnType, description)
}

// RenderScanResult 渲染扫描结果
func (r *MarkdownRenderer) RenderScanResult(address, status, summary, rawResponse string, vulnerabilities []string) string {
	var result strings.Builder

	// 合约地址作为一级标题
	result.WriteString(fmt.Sprintf("# 合约地址: %s\n\n", address))
	result.WriteString(fmt.Sprintf("**状态**: %s\n\n", status))

	// AI分析摘要
	if summary != "" {
		result.WriteString("### AI分析摘要\n\n")
		result.WriteString(fmt.Sprintf("%s\n\n", summary))
	}

	// 漏洞详情
	if len(vulnerabilities) > 0 {
		result.WriteString("### 漏洞详情\n\n")
		for i, vuln := range vulnerabilities {
			result.WriteString(fmt.Sprintf("%d. %s\n\n", i+1, vuln))
		}
	}

	// 原始AI响应
	if rawResponse != "" {
		result.WriteString("### AI原始响应\n\n")
		result.WriteString(fmt.Sprintf("```\n%s\n```\n\n", rawResponse))
	}

	return result.String()
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
