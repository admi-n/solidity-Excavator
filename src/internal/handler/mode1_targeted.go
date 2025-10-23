package handler

//
//import (
//	"bufio"
//	"context"
//	"fmt"
//	"os"
//	"strings"
//	"time"
//
//	"github.com/admi-n/solidity-Excavator/src/config"
//	"github.com/admi-n/solidity-Excavator/src/internal"
//	parser "github.com/admi-n/solidity-Excavator/src/internal/ai/parser"
//	"github.com/admi-n/solidity-Excavator/src/internal/core"
//)
//
//// RunMode1 执行 Mode1 定向扫描
//func RunMode1(cfg internal.ScanConfig) ([]parser.ScanResult, error) {
//	if cfg.Verbose {
//		fmt.Println("[Mode1] 启动定向扫描...")
//	}
//
//	// 1️⃣ 获取目标合约列表
//	targets, err := loadTargets(cfg)
//	if err != nil {
//		return nil, fmt.Errorf("[Mode1] 加载目标失败: %w", err)
//	}
//	if len(targets) == 0 {
//		return nil, fmt.Errorf("[Mode1] 未找到扫描目标")
//	}
//
//	var results []parser.ScanResult
//
//	// 2️⃣ 遍历每个合约
//	for _, contract := range targets {
//		if cfg.Verbose {
//			fmt.Printf("[Mode1] 扫描合约: %s\n", contract.Address)
//		}
//
//		// 2a. 构建 Prompt
//		prompt := buildPrompt(cfg, contract)
//		// 2b. 加载漏洞复现 EXP (占位)
//		expCode := loadExp(cfg.Strategy)
//
//		// 2c. 调用 Core 层执行扫描
//		rawOutput, err := core.Mode1Scan(contract, prompt, expCode, cfg)
//		if err != nil {
//			fmt.Printf("[Mode1] 警告: 扫描合约 %s 失败: %v\n", contract.Address, err)
//			continue
//		}
//
//		// 2d. 调用 AI Parser 将原始输出转换为 ScanResult
//		structured, err := parser.Parser.Parse(rawOutput)
//		if err != nil {
//			fmt.Printf("[Mode1] 警告: 解析合约 %s 输出失败: %v\n", contract.Address, err)
//			continue
//		}
//
//		results = append(results, structured...)
//
//		time.Sleep(100 * time.Millisecond)
//	}
//
//	if cfg.Verbose {
//		fmt.Printf("[Mode1] 扫描完成，找到 %d 个漏洞条目\n", len(results))
//	}
//
//	return results, nil
//}
//
////// loadTargets 占位函数：根据 cfg.TargetSource 获取目标合约列表
////func loadTargets(cfg internal.ScanConfig) ([]internal.Contract, error) {
////	// 如果目标来自数据库（直接拉取全表或分页），保留现有行为
////	if cfg.TargetSource == "db" {
////		dsn := os.Getenv("DATABASE_URL")
////		if dsn == "" {
////			return nil, fmt.Errorf("DATABASE_URL 未设置，无法从数据库加载目标")
////		}
////		pool, err := config.InitDB(dsn)
////		if err != nil {
////			return nil, fmt.Errorf("初始化数据库失败: %w", err)
////		}
////		// 使用短超时的 context 加载数据
////		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
////		defer cancel()
////		// limit 可根据需求调整；这里使用 1000 为示例
////		contracts, err := config.GetContracts(ctx, pool, 1000)
////		if err != nil {
////			return nil, fmt.Errorf("从数据库加载合约失败: %w", err)
////		}
////		return contracts, nil
////	}
////
////	// 如果目标来自文件（txt），则读取每行地址并用数据库按地址匹配合约信息
////	if cfg.TargetSource == "file" {
////		if cfg.TargetFile == "" {
////			return nil, fmt.Errorf("cfg.TargetFile 为空，无法读取地址文件")
////		}
////		f, err := os.Open(cfg.TargetFile)
////		if err != nil {
////			return nil, fmt.Errorf("打开地址文件失败: %w", err)
////		}
////		defer f.Close()
////
////		var addrs []string
////		scanner := bufio.NewScanner(f)
////		for scanner.Scan() {
////			line := strings.TrimSpace(scanner.Text())
////			if line == "" {
////				continue
////			}
////			addrs = append(addrs, line)
////		}
////		if err := scanner.Err(); err != nil {
////			return nil, fmt.Errorf("读取地址文件失败: %w", err)
////		}
////		if len(addrs) == 0 {
////			return nil, fmt.Errorf("地址文件中未找到任何地址")
////		}
////
////		// 使用 DB 来匹配这些地址并获取合约信息
////		dsn := os.Getenv("DATABASE_URL")
////		if dsn == "" {
////			return nil, fmt.Errorf("DATABASE_URL 未设置，无法从数据库加载目标")
////		}
////		pool, err := config.InitDB(dsn)
////		if err != nil {
////			return nil, fmt.Errorf("初始化数据库失败: %w", err)
////		}
////		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
////		defer cancel()
////		contracts, err := config.GetContractsByAddresses(ctx, pool, addrs)
////		if err != nil {
////			return nil, fmt.Errorf("按地址从数据库加载合约失败: %w", err)
////		}
////		return contracts, nil
////	}
////
////	// 默认回退：mock（方便本地测试）
////	mock := []internal.Contract{
////		{
////			Address: "0xDEADBEEF...",
////			Code:    "contract Mock { function test() public {} }",
////			// 其余字段可为空/零值
////		},
////	}
////	return mock, nil
////}
////
////// buildPrompt 占位函数
////func buildPrompt(cfg internal.ScanConfig, contract internal.Contract) string {
////	return fmt.Sprintf("请分析合约 %s 的漏洞，使用策略 %s", contract.Address, cfg.Strategy)
////}
////
////// loadExp 占位函数
////func loadExp(strategy string) string {
////	return "// 漏洞复现代码占位"
////}
