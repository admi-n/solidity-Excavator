package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadTemplate 加载指定模式和策略的 prompt 模板
func LoadTemplate(mode, strategy string) (string, error) {
	// 对于mode1，优先使用default.tmpl模板
	if mode == "mode1" {
		return LoadDefaultTemplate(mode)
	}

	// 构建模板文件路径，支持从项目根目录或src目录运行
	templatePath := filepath.Join("strategy", "prompts", mode, strategy+".tmpl")

	// 首先尝试从当前目录加载
	content, err := os.ReadFile(templatePath)
	if err != nil {
		// 如果失败，尝试从src目录加载
		srcPath := filepath.Join("src", "strategy", "prompts", mode, strategy+".tmpl")
		content, err = os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("failed to load template %s or %s: %w", templatePath, srcPath, err)
		}
	}

	return string(content), nil
}

// LoadDefaultTemplate 加载默认模板
func LoadDefaultTemplate(mode string) (string, error) {
	// 构建默认模板文件路径
	templatePath := filepath.Join("strategy", "prompts", mode, "default.tmpl")

	// 首先尝试从当前目录加载
	content, err := os.ReadFile(templatePath)
	if err != nil {
		// 如果失败，尝试从src目录加载
		srcPath := filepath.Join("src", "strategy", "prompts", mode, "default.tmpl")
		content, err = os.ReadFile(srcPath)
		if err != nil {
			return "", fmt.Errorf("failed to load default template %s or %s: %w", templatePath, srcPath, err)
		}
	}

	return string(content), nil
}

// LoadEXPCode 加载对应的 EXP 代码（如果存在）
func LoadEXPCode(mode, strategy string) (string, error) {
	// 构建 EXP 文件路径，支持从项目根目录或src目录运行
	expPath := filepath.Join("strategy", "exp_libs", mode, strategy+".t.sol")

	// 首先尝试从当前目录加载
	if _, err := os.Stat(expPath); os.IsNotExist(err) {
		// 如果失败，尝试从src目录加载
		srcPath := filepath.Join("src", "strategy", "exp_libs", mode, strategy+".t.sol")
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			return "", nil // EXP 不存在不算错误
		}
		expPath = srcPath
	}

	// 读取 EXP 代码
	content, err := os.ReadFile(expPath)
	if err != nil {
		return "", fmt.Errorf("failed to load EXP code %s: %w", expPath, err)
	}

	return string(content), nil
}

// LoadInputFile 加载指定的输入文件
func LoadInputFile(inputFile string) (string, error) {
	if inputFile == "" {
		return "", nil
	}

	// 如果输入的是文件名（不包含路径），则在默认目录中查找
	if !strings.Contains(inputFile, "/") && !strings.Contains(inputFile, "\\") {
		// 首先尝试从当前目录加载
		if _, err := os.Stat(inputFile); os.IsNotExist(err) {
			// 如果失败，尝试从 src/strategy/exp_libs/mode1/ 目录加载
			defaultPath := filepath.Join("src", "strategy", "exp_libs", "mode1", inputFile)
			if _, err := os.Stat(defaultPath); err == nil {
				inputFile = defaultPath
			}
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return "", fmt.Errorf("input file not found: %s", inputFile)
	}

	// 读取文件内容
	content, err := os.ReadFile(inputFile)
	if err != nil {
		return "", fmt.Errorf("failed to load input file %s: %w", inputFile, err)
	}

	// 根据文件扩展名处理不同格式
	ext := filepath.Ext(inputFile)
	switch ext {
	case ".toml":
		processedContent := processTOMLFile(string(content))
		return processedContent, nil
	case ".sol":
		processedContent := processMarkers(string(content))
		return processedContent, nil
	default:
		// 默认按TOML处理
		processedContent := processTOMLFile(string(content))
		return processedContent, nil
	}
}

// processTOMLFile 处理TOML文件，提取漏洞合约源码和复现代码
func processTOMLFile(content string) string {
	var result strings.Builder

	// 查找[漏洞合约源码]部分
	vulnStart := strings.Index(content, "[漏洞合约源码]")
	if vulnStart != -1 {
		// 查找code = """开始位置
		codeStart := strings.Index(content[vulnStart:], "code = \"\"\"")
		if codeStart != -1 {
			codeStart += vulnStart + len("code = \"\"\"")
			// 查找结束的"""
			codeEnd := strings.Index(content[codeStart:], "\"\"\"")
			if codeEnd != -1 {
				vulnCode := strings.TrimSpace(content[codeStart : codeStart+codeEnd])
				result.WriteString("==漏洞合约源码==\n")
				result.WriteString(vulnCode)
				result.WriteString("\n\n")
			}
		}
	}

	// 查找[Foundry复现代码]部分
	foundryStart := strings.Index(content, "[Foundry复现代码]")
	if foundryStart != -1 {
		// 查找code = """开始位置
		codeStart := strings.Index(content[foundryStart:], "code = \"\"\"")
		if codeStart != -1 {
			codeStart += foundryStart + len("code = \"\"\"")
			// 查找结束的"""
			codeEnd := strings.Index(content[codeStart:], "\"\"\"")
			if codeEnd != -1 {
				foundryCode := strings.TrimSpace(content[codeStart : codeStart+codeEnd])
				result.WriteString("==Foundry复现代码==\n")
				result.WriteString(foundryCode)
				result.WriteString("\n")
			}
		}
	}

	// 如果没有找到TOML结构，直接返回原内容
	if result.Len() == 0 {
		return content
	}

	return result.String()
}

// processMarkers 处理文件中的标记，确保格式正确
func processMarkers(content string) string {
	// 确保标记格式正确
	content = strings.ReplaceAll(content, "==漏洞合约源码==", "==漏洞合约源码==")
	content = strings.ReplaceAll(content, "==Foundry复现代码==", "==Foundry复现代码==")

	// 如果文件没有标记，添加默认标记
	if !strings.Contains(content, "==漏洞合约源码==") && !strings.Contains(content, "==Foundry复现代码==") {
		// 这是一个没有标记的文件，直接返回
		return content
	}

	return content
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
