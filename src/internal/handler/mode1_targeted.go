package handler

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/admi-n/solidity-Excavator/src/config"
	"github.com/admi-n/solidity-Excavator/src/internal"
	"github.com/admi-n/solidity-Excavator/src/internal/ai"
	"github.com/admi-n/solidity-Excavator/src/internal/ai/parser"
	"github.com/admi-n/solidity-Excavator/src/internal/download"
	"github.com/admi-n/solidity-Excavator/src/strategy/prompts"
)

// // RunMode1Targeted 执行 Mode1 定向扫描
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
	switch strings.ToLower(cfg.TargetSource) {
	case "db":
		targetAddresses, err = getAddressesFromDB(db, cfg.BlockRange)
		if err != nil {
			return fmt.Errorf("从数据库获取地址失败: %w", err)
		}
	case "file", "filepath":
		targetAddresses, err = getAddressesFromFile(cfg.TargetFile)
		if err != nil {
			return fmt.Errorf("从文件获取地址失败: %w", err)
		}
	case "contract", "address", "single":
		if strings.TrimSpace(cfg.TargetAddress) == "" {
			return fmt.Errorf("缺少目标合约地址: -t-address")
		}
		targetAddresses = []string{strings.TrimSpace(cfg.TargetAddress)}
	default:
		return fmt.Errorf("不支持的目标源: %s", cfg.TargetSource)
	}

	if len(targetAddresses) == 0 {
		fmt.Println("⚠️  没有找到可扫描的合约")
		return nil
	}

	fmt.Printf("📋 共找到 %d 个目标合约\n", len(targetAddresses))

	// 6. 创建下载器（用于获取合约代码）
	downloader, err := download.NewDownloader(db, cfg.Proxy)
	if err != nil {
		return fmt.Errorf("创建下载器失败: %w", err)
	}
	defer func() {
		if downloader != nil && downloader.Client != nil {
			downloader.Client.Close()
		}
	}()

	// 7. 处理每个合约
	results := make([]*ScanResult, 0, len(targetAddresses))
	successCount := 0
	failCount := 0

	for i, address := range targetAddresses {
		fmt.Printf("\n[%d/%d] 处理合约: %s\n", i+1, len(targetAddresses), address)

		// 7.1 获取合约代码
		contractCode, err := getOrDownloadContract(ctx, db, downloader, address)
		if err != nil {
			fmt.Printf("⚠️  获取合约代码失败: %v，跳过\n", err)
			failCount++
			continue
		}

		// 检查是否为字节码（以 0x 开头且全是十六进制）
		if isOnlyBytecode(contractCode) {
			fmt.Println("  ⏭️  合约未开源（仅字节码），跳过分析")
			failCount++
			continue
		}

		// 7.2 构建 prompt（使用专门的 Mode1 构建器）
		var prompt string
		if cfg.ExpFile != "" {
			// 尝试读取 exp 文件
			expBs, _ := os.ReadFile(cfg.ExpFile)
			expCode := strings.TrimSpace(string(expBs))
			if expCode != "" {
				prompt = prompts.BuildMode1Prompt(address, contractCode, cfg.Strategy, expCode)
			} else {
				prompt = prompts.BuildPrompt(promptTemplate, map[string]string{
					"ContractAddress": address,
					"ContractCode":    contractCode,
					"Strategy":        cfg.Strategy,
				})
			}
		} else {
			prompt = prompts.BuildPrompt(promptTemplate, map[string]string{
				"ContractAddress": address,
				"ContractCode":    contractCode,
				"Strategy":        cfg.Strategy,
			})
		}

		// 7.3 调用 AI 分析
		analysisResult, err := aiManager.AnalyzeContract(ctx, contractCode, prompt)
		if err != nil {
			fmt.Printf("⚠️  AI 分析失败: %v，跳过\n", err)
			failCount++
			continue
		}

		// 7.4 保存结果
		scanResult := &ScanResult{
			Address:        address,
			AnalysisResult: analysisResult,
			Timestamp:      time.Now(),
			Mode:           cfg.Mode,
			Strategy:       cfg.Strategy,
		}
		results = append(results, scanResult)
		successCount++

		// 打印摘要
		fmt.Printf("%s\n", strings.Repeat("=", 50))
		printVulnerabilitySummary(scanResult)
		fmt.Printf("%s\n", strings.Repeat("=", 50))

		// 避免请求过快
		time.Sleep(100 * time.Millisecond)
	}

	// 8. 打印总结
	fmt.Printf("\n%s\n", strings.Repeat("=", 50))
	fmt.Printf("✅ 扫描完成！\n")
	fmt.Printf("   - 总合约数: %d\n", len(targetAddresses))
	fmt.Printf("   - 成功分析: %d\n", successCount)
	fmt.Printf("   - 失败/跳过: %d\n", failCount)
	fmt.Printf("   - 发现漏洞的合约: %d\n", countVulnerableContracts(results))
	fmt.Printf("%s\n\n", strings.Repeat("=", 50))

	// 9. 生成报告
	if len(results) > 0 {
		if err := generateReport(results, cfg); err != nil {
			return fmt.Errorf("生成报告失败: %w", err)
		}
	}

	return nil
}

// isOnlyBytecode 检查是否为纯字节码（未开源）
func isOnlyBytecode(code string) bool {
	code = strings.TrimSpace(code)
	if len(code) < 10 {
		return true
	}
	if !strings.HasPrefix(code, "0x") {
		// 如果不是 0x 开头，认为是源码
		return false
	}
	for _, c := range code[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// getOrDownloadContract 从数据库获取合约代码，如果不存在则下载
func getOrDownloadContract(ctx context.Context, db *sql.DB, downloader *download.Downloader, address string) (string, error) {
	// 先尝试从数据库获取（注意：字段名是 contract）
	var contractCode string
	query := "SELECT contract FROM contracts WHERE address = ? AND contract IS NOT NULL AND contract != ''"
	err := db.QueryRow(query, address).Scan(&contractCode)
	if err == nil && contractCode != "" {
		fmt.Println("  ✓ 从数据库读取合约代码")
		return contractCode, nil
	}

	// 数据库中不存在，尝试下载（下载器会把源码写入 DB，如果可用）
	fmt.Println("  ↓ 合约不在数据库中，正在下载...")
	if err := downloader.DownloadContractsByAddresses(ctx, []string{address}, ""); err != nil {
		// 回退为从链上读取字节码
		codeBytes, rcErr := downloader.Client.CodeAt(ctx, common.HexToAddress(address), nil)
		if rcErr != nil {
			return "", fmt.Errorf("下载合约失败: %v, 且回退获取字节码失败: %w", err, rcErr)
		}
		return fmt.Sprintf("0x%x", codeBytes), nil
	}

	// 尝试再次从数据库读取
	err = db.QueryRow(query, address).Scan(&contractCode)
	if err == nil && contractCode != "" {
		return contractCode, nil
	}

	return "", fmt.Errorf("未能获取合约源码，仅存在字节码或不存在")
}

// getAddressesFromDB 从数据库读取地址列表，支持按区间查询
func getAddressesFromDB(db *sql.DB, blockRange *internal.BlockRange) ([]string, error) {
	var query string
	var args []interface{}

	// 构建基础查询条件
	baseConditions := "isopensource = 1 AND contract IS NOT NULL AND contract != ''"

	if blockRange != nil {
		// 如果有区块范围限制，添加区块条件
		query = fmt.Sprintf(`SELECT DISTINCT address FROM contracts WHERE %s AND createblock BETWEEN ? AND ? LIMIT 1000`, baseConditions)
		args = []interface{}{blockRange.Start, blockRange.End}
	} else {
		// 默认返回前 1000 个开源合约
		query = fmt.Sprintf(`SELECT DISTINCT address FROM contracts WHERE %s LIMIT 1000`, baseConditions)
		args = []interface{}{}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	addrs := make([]string, 0)
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			return nil, err
		}
		addrs = append(addrs, strings.TrimSpace(a))
	}
	return addrs, nil
}

// getAddressesFromFile 从文件获取地址列表
func getAddressesFromFile(filepathStr string) ([]string, error) {
	if strings.TrimSpace(filepathStr) == "" {
		return nil, fmt.Errorf("文件路径为空")
	}
	bs, err := os.ReadFile(filepathStr)
	if err != nil {
		return nil, err
	}
	text := string(bs)
	lines := strings.Split(text, "\n")
	addrs := make([]string, 0, len(lines))
	for _, l := range lines {
		line := strings.TrimSpace(l)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}
		// 支持以逗号或空格分隔的多字段，取第一个字段
		fields := strings.FieldsFunc(line, func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
		if len(fields) == 0 {
			continue
		}
		addrs = append(addrs, strings.TrimSpace(fields[0]))
	}
	return addrs, nil
}

// ScanResult 扫描结果结构
type ScanResult struct {
	Address        string
	AnalysisResult *parser.AnalysisResult
	Timestamp      time.Time
	Mode           string
	Strategy       string
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
		severityEmoji := getSeverityEmoji(vuln.Severity)
		fmt.Printf("    %d. %s [%s] %s\n",
			i+1, severityEmoji, vuln.Severity, vuln.Type)
		if vuln.Description != "" && len(vuln.Description) < 200 {
			fmt.Printf("       描述: %s\n", vuln.Description)
		}
	}
}

// getSeverityEmoji 根据严重性返回对应的表情符号
func getSeverityEmoji(severity string) string {
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

// countVulnerableContracts 统计有漏洞的合约数量
func countVulnerableContracts(results []*ScanResult) int {
	count := 0
	for _, r := range results {
		if r.AnalysisResult != nil && len(r.AnalysisResult.Vulnerabilities) > 0 {
			count++
		}
	}
	return count
}

// generateReport 生成扫描报告并写入文件
func generateReport(results []*ScanResult, cfg internal.ScanConfig) error {
	fmt.Println("\n📄 生成扫描报告...")
	// 以模式和时间生成文件名
	reportFile := fmt.Sprintf("scan_report_%s_%d.txt", strings.ReplaceAll(cfg.Mode, " ", "_"), time.Now().Unix())
	content := generateTextReport(results, cfg)
	if err := writeReportToFile(reportFile, content); err != nil {
		return err
	}
	fmt.Printf("✅ 报告已保存: %s\n", reportFile)
	return nil
}

// generateTextReport 生成文本格式报告
func generateTextReport(results []*ScanResult, cfg internal.ScanConfig) string {
	var sb strings.Builder

	sb.WriteString("========================================\n")
	sb.WriteString("    Solidity Excavator 扫描报告\n")
	sb.WriteString("========================================\n\n")
	sb.WriteString(fmt.Sprintf("扫描模式: %s\n", cfg.Mode))
	sb.WriteString(fmt.Sprintf("策略: %s\n", cfg.Strategy))
	sb.WriteString(fmt.Sprintf("AI 提供商: %s\n", cfg.AIProvider))
	sb.WriteString(fmt.Sprintf("扫描时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	sb.WriteString("----------------------------------------\n")
	sb.WriteString("扫描统计\n")
	sb.WriteString("----------------------------------------\n")
	sb.WriteString(fmt.Sprintf("总合约数: %d\n", len(results)))
	sb.WriteString(fmt.Sprintf("存在漏洞: %d\n", countVulnerableContracts(results)))

	// 按严重性统计
	severityCounts := make(map[string]int)
	for _, r := range results {
		if r.AnalysisResult != nil {
			for _, v := range r.AnalysisResult.Vulnerabilities {
				severityCounts[v.Severity]++
			}
		}
	}

	sb.WriteString("\n漏洞严重性分布:\n")
	for _, severity := range []string{"Critical", "High", "Medium", "Low"} {
		if count, ok := severityCounts[severity]; ok && count > 0 {
			sb.WriteString(fmt.Sprintf("  %s: %d\n", severity, count))
		}
	}

	sb.WriteString("\n========================================\n")
	sb.WriteString("详细结果\n")
	sb.WriteString("========================================\n\n")

	for i, result := range results {
		sb.WriteString(fmt.Sprintf("[%d] 合约地址: %s\n", i+1, result.Address))
		sb.WriteString(fmt.Sprintf("    扫描时间: %s\n", result.Timestamp.Format("2006-01-02 15:04:05")))

		if result.AnalysisResult == nil {
			sb.WriteString("    状态: 分析失败\n\n")
			continue
		}

		vulnCount := len(result.AnalysisResult.Vulnerabilities)
		sb.WriteString(fmt.Sprintf("    状态: ⚠️ 发现 %d 个漏洞\n", vulnCount))
		if result.AnalysisResult.RiskScore > 0 {
			sb.WriteString(fmt.Sprintf("    风险评分: %.1f/10\n", result.AnalysisResult.RiskScore))
		}

		sb.WriteString("\n    漏洞详情:\n")
		for j, vuln := range result.AnalysisResult.Vulnerabilities {
			sb.WriteString(fmt.Sprintf("    %d. [%s] %s\n", j+1, vuln.Severity, vuln.Type))
			if vuln.Description != "" {
				sb.WriteString(fmt.Sprintf("       描述: %s\n", vuln.Description))
			}
			if vuln.Location != "" {
				sb.WriteString(fmt.Sprintf("       位置: %s\n", vuln.Location))
			}
			if vuln.Remediation != "" {
				sb.WriteString(fmt.Sprintf("       修复建议: %s\n", vuln.Remediation))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("----------------------------------------\n\n")
	}

	return sb.String()
}

// writeReportToFile 将报告写入文件
func writeReportToFile(filename, content string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建报告文件失败: %w", err)
	}
	defer f.Close()

	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("写入报告失败: %w", err)
	}
	return nil
}
