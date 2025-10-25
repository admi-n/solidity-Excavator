package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/admi-n/solidity-Excavator/src/config"
	"github.com/admi-n/solidity-Excavator/src/internal"
	"github.com/admi-n/solidity-Excavator/src/internal/download"
	"github.com/admi-n/solidity-Excavator/src/internal/handler"
)

// ExecuteDownload 执行下载命令
func ExecuteDownload(cfg *CLIConfig) error {
	fmt.Println("🚀 启动合约下载器...")

	// 初始化 MySQL 数据库连接
	fmt.Println("📊 正在连接 MySQL 数据库...")
	db, err := config.InitDB()
	if err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}
	defer db.Close()
	fmt.Println("✅ 数据库连接成功!")

	// 创建下载器
	fmt.Println("🔗 正在创建下载器...")
	dl, err := download.NewDownloader(db, cfg.Proxy)
	if err != nil {
		return fmt.Errorf("创建下载器失败: %w", err)
	}
	defer dl.Close()

	// 创建上下文
	ctx := context.Background()

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("开始同步合约数据...")
	fmt.Println(strings.Repeat("=", 50) + "\n")

	// 如果用户传入 -file，则从该文件读取地址并逐条下载
	if cfg.DownloadFile != "" {
		// 读取文件中的地址（每行一个），去重并传给下载器
		fpath := cfg.DownloadFile
		f, err := os.Open(fpath)
		if err != nil {
			return fmt.Errorf("打开地址文件失败: %w", err)
		}
		scanner := bufio.NewScanner(f)
		var addrs []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// 跳过空行和注释行
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// 基本验证：以太坊地址应该是42个字符，以0x开头
			if len(line) == 42 && strings.HasPrefix(line, "0x") {
				addrs = append(addrs, line)
			} else {
				fmt.Printf("⚠️  跳过无效地址: %s\n", line)
			}
		}
		f.Close()
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("读取地址文件失败: %w", err)
		}
		if len(addrs) == 0 {
			return fmt.Errorf("地址文件为空: %s", fpath)
		}

		// 将未下载成功的地址写入默认失败文件 eoferror.txt
		failLog := "eoferror.txt"
		fmt.Printf("🔁 正在根据 %s 重试 %d 个地址，失败将记录到 %s\n", fpath, len(addrs), failLog)
		if err := dl.DownloadContractsByAddresses(ctx, addrs, failLog); err != nil {
			return fmt.Errorf("按地址下载失败: %w", err)
		}

		fmt.Println("\n🎉 地址下载完成!")
		return nil
	}

	// 如果没有指定文件，则按区块范围或从上次继续下载
	if cfg.DownloadRange != nil {
		start := cfg.DownloadRange.Start
		end := cfg.DownloadRange.End
		if end == ^uint64(0) {
			return fmt.Errorf("下载范围的结束区块不能为空")
		}
		fmt.Printf("📥 下载指定区块范围: %d 到 %d\n", start, end)
		if err := dl.DownloadBlockRange(ctx, start, end); err != nil {
			return fmt.Errorf("下载失败: %w", err)
		}
	} else {
		fmt.Println("📥 从上次下载位置继续...")
		if err := dl.DownloadFromLast(ctx); err != nil {
			return fmt.Errorf("从上次继续下载失败: %w", err)
		}
	}

	fmt.Println("\n🎉 下载任务完成!")
	return nil
}

// ExecuteScan 执行扫描命令
func ExecuteScan(cfg *CLIConfig) error {
	// 加载配置文件
	if err := config.LoadSettings("src/config/settings.yaml"); err != nil {
		fmt.Printf("⚠️  警告: 无法加载配置文件: %v\n", err)
		fmt.Println("将尝试从环境变量读取配置...")
	}

	// 将 CLIConfig 映射到 internal.ScanConfig
	internalCfg := internal.ScanConfig{
		AIProvider:    cfg.AIProvider,
		Mode:          cfg.Mode,
		Strategy:      cfg.Strategy,
		TargetSource:  cfg.TargetSource,
		TargetFile:    cfg.TargetFile,
		TargetAddress: cfg.TargetAddress,
		Chain:         cfg.Chain,
		Concurrency:   cfg.Concurrency,
		Verbose:       cfg.Verbose,
		Timeout:       cfg.Timeout,
		InputFile:     cfg.InputFile,
		Proxy:         cfg.Proxy,
	}
	if cfg.BlockRange != nil {
		internalCfg.BlockRange = &internal.BlockRange{
			Start: cfg.BlockRange.Start,
			End:   cfg.BlockRange.End,
		}
	}

	// 根据模式分派到相应处理器
	switch cfg.Mode {
	case "mode1":
		fmt.Println("🎯 启动 Mode1（定向扫描）处理器...")
		return handler.RunMode1Targeted(internalCfg)

	case "mode2":
		fmt.Println("🔍 启动 Mode2（模糊扫描）处理器...")
		return fmt.Errorf("Mode2 暂未实现")

	case "mode3":
		fmt.Println("🌐 启动 Mode3（通用扫描）处理器...")
		return fmt.Errorf("Mode3 暂未实现")

	default:
		return fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}
}

// Execute 执行主命令逻辑
func Execute(cfg *CLIConfig) error {
	// 下载模式优先
	if cfg.Download {
		return ExecuteDownload(cfg)
	}

	// 非下载模式：正常的扫描流程
	if cfg.Verbose {
		fmt.Printf("使用配置运行 Excavator: %+v\n", cfg)
	}

	return ExecuteScan(cfg)
}
