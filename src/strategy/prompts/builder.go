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
