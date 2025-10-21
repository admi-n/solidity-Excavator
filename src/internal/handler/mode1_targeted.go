package handler

import (
	"fmt"
	"time"

	"github.com/admi-n/solidity-Excavator/src/internal"
	parser "github.com/admi-n/solidity-Excavator/src/internal/ai/parser"
	"github.com/admi-n/solidity-Excavator/src/internal/core"
)

// RunMode1 执行 Mode1 定向扫描
func RunMode1(cfg internal.ScanConfig) ([]parser.ScanResult, error) {
	if cfg.Verbose {
		fmt.Println("[Mode1] 启动定向扫描...")
	}

	// 1️⃣ 获取目标合约列表
	targets, err := loadTargets(cfg)
	if err != nil {
		return nil, fmt.Errorf("[Mode1] 加载目标失败: %w", err)
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("[Mode1] 未找到扫描目标")
	}

	var results []parser.ScanResult

	// 2️⃣ 遍历每个合约
	for _, contract := range targets {
		if cfg.Verbose {
			fmt.Printf("[Mode1] 扫描合约: %s\n", contract.Address)
		}

		// 2a. 构建 Prompt
		prompt := buildPrompt(cfg, contract)
		// 2b. 加载漏洞复现 EXP (占位)
		expCode := loadExp(cfg.Strategy)

		// 2c. 调用 Core 层执行扫描
		rawOutput, err := core.Mode1Scan(contract, prompt, expCode, cfg)
		if err != nil {
			fmt.Printf("[Mode1] 警告: 扫描合约 %s 失败: %v\n", contract.Address, err)
			continue
		}

		// 2d. 调用 AI Parser 将原始输出转换为 ScanResult
		structured, err := parser.Parser.Parse(rawOutput)
		if err != nil {
			fmt.Printf("[Mode1] 警告: 解析合约 %s 输出失败: %v\n", contract.Address, err)
			continue
		}

		results = append(results, structured...)

		time.Sleep(100 * time.Millisecond)
	}

	if cfg.Verbose {
		fmt.Printf("[Mode1] 扫描完成，找到 %d 个漏洞条目\n", len(results))
	}

	return results, nil
}

// loadTargets 占位函数：根据 cfg.TargetSource 获取目标合约列表
func loadTargets(cfg internal.ScanConfig) ([]internal.Contract, error) {
	// TODO: 根据 cfg.TargetSource == "db" 或 "file" 加载真实数据
	mock := []internal.Contract{
		{
			Address: "0xDEADBEEF...",
			Code:    "contract Mock { function test() public {} }",
		},
	}
	return mock, nil
}

// buildPrompt 占位函数
func buildPrompt(cfg internal.ScanConfig, contract internal.Contract) string {
	return fmt.Sprintf("请分析合约 %s 的漏洞，使用策略 %s", contract.Address, cfg.Strategy)
}

// loadExp 占位函数
func loadExp(strategy string) string {
	return "// 漏洞复现代码占位"
}
