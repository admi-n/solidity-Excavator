package handler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/admi-n/solidity-Excavator/src/config"
	"github.com/admi-n/solidity-Excavator/src/internal"
	"github.com/admi-n/solidity-Excavator/src/internal/ai"
	"github.com/admi-n/solidity-Excavator/src/internal/download"
	"github.com/admi-n/solidity-Excavator/src/strategy/prompts"
)

// RunMode1Targeted 执行 Mode1 定向扫描
func RunMode1Targeted(cfg internal.ScanConfig) error {
	fmt.Println("🎯 启动 Mode1 定向漏洞扫描...")

	// 1. 初始化数据库
	db, err := config.InitDB()
	if err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer db.Close()

	// 2. 创建 AI 管理器
	aiManager, err := ai.NewManager(ai.ManagerConfig{
		Provider:       cfg.AIProvider,
		Timeout:        cfg.Timeout,
		RequestsPerMin: 20, // 每分钟 20 个请求
	})
	if err != nil {
		return fmt.Errorf("创建 AI 管理器失败: %w", err)
	}
	defer aiManager.Close()

	// 3. 测试 AI 连接
	ctx := context.Background()
	if err := aiManager.TestConnection(ctx); err != nil {
		return fmt.Errorf("AI 连接测试失败: %w", err)
	}

	// 4. 加载 prompt 模板
	promptTemplate, err := prompts.LoadTemplate(cfg.Mode, cfg.Strategy)
	if err != nil {
		return fmt.Errorf("加载 prompt 模板失败: %w", err)
	}

	// 5. 获取目标合约地址
	var targetAddresses []string
	switch cfg.TargetSource {
	case "db":
		// 从数据库获取地址
		targetAddresses, err = getAddressesFromDB(db, cfg.BlockRange)
		if err != nil {
			return fmt.Errorf("从数据库获取地址失败: %w", err)
		}
	case "file":
		// 从文件获取地址
		targetAddresses, err = getAddressesFromFile(cfg.TargetFile)
		if err != nil {
			return fmt.Errorf("从文件获取地址失败: %w", err)
		}
	default:
		return fmt.Errorf("不支持的目标源: %s", cfg.TargetSource)
	}

	fmt.Printf("📋 共找到 %d 个目标合约\n", len(targetAddresses))

	// 6. 创建下载器（用于获取合约代码）
	downloader, err := download.NewDownloader(db, "")
	if err != nil {
		return fmt.Errorf("创建下载器失败: %w", err)
	}
	defer downloader.Close()

	// 7. 处理每个合约
	results := make([]*ScanResult, 0)
	for i, address := range targetAddresses {
		fmt.Printf("\n[%d/%d] 处理合约: %s\n", i+1, len(targetAddresses), address)

		// 7.1 获取合约代码
		contractCode, err := getOrDownloadContract(ctx, db, downloader, address)
		if err != nil {
			fmt.Printf("⚠️  获取合约代码失败: %v，跳过\n", err)
			continue
		}

		// 7.2 构建 prompt
		prompt := prompts.BuildPrompt(promptTemplate, map[string]string{
			"ContractAddress": address,
			"ContractCode":    contractCode,
			"Strategy":        cfg.Strategy,
		})

		// 7.3 调用 AI 分析
		analysisResult, err := aiManager.AnalyzeContract(ctx, contractCode, prompt)
		if err != nil {
			fmt.Printf("⚠️  AI 分析失败: %v，跳过\n", err)
			continue
		}

		// 7.4 保存结果
		scanResult := &ScanResult{
			Address:        address,
			AnalysisResult: analysisResult,
			Timestamp:      time.Now(),
		}
		results = append(results, scanResult)

		// 打印漏洞摘要
		printVulnerabilitySummary(scanResult)
	}

	// 8. 生成报告
	fmt.Printf("\n✅ 扫描完成！共分析 %d 个合约\n", len(results))
	if err := generateReport(results, cfg); err != nil {
		return fmt.Errorf("生成报告失败: %w", err)
	}

	return nil
}

// getOrDownloadContract 从数据库获取合约代码，如果不存在则下载
func getOrDownloadContract(ctx context.Context, db *sql.DB, downloader *download.Downloader, address string) (string, error) {
	// 先尝试从数据库获取
	var sourceCode string
	query := "SELECT source_code FROM contracts WHERE address = ? AND source_code IS NOT NULL AND source_code != ''"
	err := db.QueryRow(query, address).Scan(&sourceCode)

	if err == nil && sourceCode != "" {
		fmt.Println("  ✓ 从数据库读取合约代码")
		return sourceCode, nil
	}

	// 数据库中不存在，尝试下载
	fmt.Println("  ↓ 合约不在数据库中，正在下载...")

	// 使用下载器获取合约
	if err := downloader.DownloadContractsByAddresses(ctx, []string{address}, ""); err != nil {
		return "", fmt.Errorf("下载合约失败: %w", err)
	}

	// 再次从数据库读取
	err = db.QueryRow(query, address).Scan(&sourceCode)
	if err != nil {
		return "", fmt.Errorf("下载后仍无法获取合约代码: %w", err)
	}

	fmt.Println("  ✓ 下载并保存合约代码")
	return sourceCode, nil
}

// getAddressesFromDB 从数据库获取地址列表
func getAddressesFromDB(db *sql.DB, blockRange *internal.BlockRange) ([]string, error) {
	var query string
	var args []interface{}

	if blockRange != nil {
		query = `SELECT DISTINCT address FROM contracts 
				 WHERE block_number >= ? AND block_number <= ? 
				 AND source_code IS NOT NULL AND source_code != ''
				 LIMIT 1000`
		args = []interface{}{blockRange.Start, blockRange.End}
	} else {
		query = `SELECT DISTINCT address FROM contracts 
				 WHERE source_code IS NOT NULL AND source_code != ''
				 LIMIT 1000`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []string
	for rows.Next() {
		var addr string
		if err := rows.Scan(&addr); err != nil {
			return nil, err
		}
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// getAddressesFromFile 从文件获取地址列表
func getAddressesFromFile(filepath string) ([]string, error) {
	// TODO: 实现从 YAML 文件读取地址
	// 可以使用 yaml.Unmarshal 解析文件
	return nil, fmt.Errorf("从文件读取地址功能待实现")
}

// ScanResult 扫描结果结构
type ScanResult struct {
	Address        string
	AnalysisResult *ai.AnalysisResult
	Timestamp      time.Time
}

// printVulnerabilitySummary 打印漏洞摘要
func printVulnerabilitySummary(result *ScanResult) {
	if result.AnalysisResult == nil {
		return
	}

	vulnCount := len(result.AnalysisResult.Vulnerabilities)
	if vulnCount == 0 {
		fmt.Println("  ✅ 未发现漏洞")
		return
	}

	fmt.Printf("  ⚠️  发现 %d 个潜在漏洞:\n", vulnCount)
	for i, vuln := range result.AnalysisResult.Vulnerabilities {
		fmt.Printf("    %d. [%s] %s - %s\n",
			i+1, vuln.Severity, vuln.Type, vuln.Description)
	}
}

// generateReport 生成扫描报告
func generateReport(results []*ScanResult, cfg internal.ScanConfig) error {
	// TODO: 调用 report 包生成报告
	fmt.Println("\n📄 生成扫描报告...")
	return nil
}
