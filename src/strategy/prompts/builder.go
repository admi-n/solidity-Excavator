package prompts

import (
	"fmt"
	"strings"
	"text/template"
)

// BuildPrompt 使用模板和变量构建最终的 prompt
func BuildPrompt(templateContent string, variables map[string]string) string {
	tmpl, err := template.New("prompt").Parse(templateContent)
	if err != nil {
		return fmt.Sprintf("模板解析失败: %v\n原始模板:\n%s", err, templateContent)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, variables); err != nil {
		return fmt.Sprintf("模板执行失败: %v\n原始模板:\n%s", err, templateContent)
	}

	return result.String()
}

// BuildMode1Prompt 构建 Mode1 专用的 prompt，包含 EXP 代码
func BuildMode1Prompt(contractAddress, contractCode, strategy, expCode string) string {
	// 加载基础模板
	templateContent, err := LoadTemplate("mode1", strategy)
	if err != nil {
		// 如果模板加载失败，使用默认模板
		templateContent = getDefaultMode1Template()
	}

	// 构建变量映射
	variables := map[string]string{
		"ContractAddress":    contractAddress,
		"ContractCode":       contractCode,
		"Strategy":           strategy,
		"hourglassvul.t.sol": expCode, // 将 EXP 代码注入到模板中
	}

	return BuildPrompt(templateContent, variables)
}

// getDefaultMode1Template 获取默认的 Mode1 模板
func getDefaultMode1Template() string {
	return `You are an expert Solidity security auditor specializing in DeFi vulnerabilities.

**Target Contract:**
Contract Address: {{.ContractAddress}}

**Reproducing the historical attack using the foundry's test simulation:**
{{.hourglassvul.t.sol}}

**Vulnerability Pattern: {{.Strategy}}**

**Analysis Task:**
Carefully examine the following contract and determine if it contains similar vulnerability patterns.

Contract Code:
{{.ContractCode}}`
}
