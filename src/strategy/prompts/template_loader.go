package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadTemplate 加载指定模式和策略的 prompt 模板
func LoadTemplate(mode, strategy string) (string, error) {
	// 构建模板文件路径
	templatePath := filepath.Join("strategy", "prompts", mode, strategy+".tmpl")

	// 读取模板文件
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load template %s: %w", templatePath, err)
	}

	return string(content), nil
}

// LoadEXPCode 加载对应的 EXP 代码（如果存在）
func LoadEXPCode(mode, strategy string) (string, error) {
	// 构建 EXP 文件路径
	expPath := filepath.Join("strategy", "exp_libs", mode, strategy+".t.sol")

	// 检查文件是否存在
	if _, err := os.Stat(expPath); os.IsNotExist(err) {
		return "", nil // EXP 不存在不算错误
	}

	// 读取 EXP 代码
	content, err := os.ReadFile(expPath)
	if err != nil {
		return "", fmt.Errorf("failed to load EXP code %s: %w", expPath, err)
	}

	return string(content), nil
}

// ListStrategies 列出指定模式下所有可用的策略
func ListStrategies(mode string) ([]string, error) {
	promptsDir := filepath.Join("strategy", "prompts", mode)

	// 检查目录是否存在
	if _, err := os.Stat(promptsDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("prompts directory not found for mode %s", mode)
	}

	// 读取目录中的所有 .tmpl 文件
	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read prompts directory: %w", err)
	}

	var strategies []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tmpl") {
			// 移除 .tmpl 后缀
			strategyName := strings.TrimSuffix(entry.Name(), ".tmpl")
			strategies = append(strategies, strategyName)
		}
	}

	if len(strategies) == 0 {
		return nil, fmt.Errorf("no strategies found in %s", promptsDir)
	}

	return strategies, nil
}
