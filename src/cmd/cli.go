package cmd

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/admi-n/solidity-Excavator/src/internal/handler"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/admi-n/solidity-Excavator/src/config"
	"github.com/admi-n/solidity-Excavator/src/internal"
	"github.com/admi-n/solidity-Excavator/src/internal/download"
)

// Reporter 先不写

// CLIConfig 保存解析好的 CLI 选项以及供扫描器使用的规范化字段。
type CLIConfig struct {
	AIProvider    string // 例如 chatgpt5
	Mode          string // mode1 | mode2 | mode3
	Strategy      string // 例如 hourglass-vul 或 "all"
	TargetSource  string // "db" 或 "file" 或 "contract" - 从哪里获取目标列表
	TargetFile    string // 包含地址/批次的 YAML 路径
	TargetAddress string // 单个合约地址，当 -t=contract 时使用
	BlockRange    *BlockRange
	Chain         string // eth | bsc | arb
	Concurrency   int
	Verbose       bool
	Timeout       time.Duration

	// 下载相关配置
	Download      bool        // -d 启动下载流程
	DownloadRange *BlockRange // -d-range 指定下载区块范围（格式 start-end），为空表示从上次继续下载
	DownloadFile  string      // -file 指定包含地址的 txt 文件（每行一个地址），用于重试下载

	Proxy string // 新增：HTTP 代理 (例如 http://127.0.0.1:7897)
}

// BlockRange 简单的起止区块范围结构
type BlockRange struct {
	Start uint64
	End   uint64
}

func (b *BlockRange) String() string {
	if b == nil {
		return ""
	}
	return fmt.Sprintf("%d-%d", b.Start, b.End)
}

// parseBlockRange 解析类似 "1-220234" 或 "1000-"（开放结束）的字符串并返回 BlockRange。
func parseBlockRange(s string) (*BlockRange, error) {
	if strings.TrimSpace(s) == "" {
		return nil, nil
	}
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return nil, errors.New("invalid block range format, expected start-end")
	}
	startStr := strings.TrimSpace(parts[0])
	endStr := strings.TrimSpace(parts[1])
	var br BlockRange
	if startStr == "" {
		return nil, errors.New("start block required")
	}
	start, err := strconv.ParseUint(startStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid start block: %w", err)
	}
	br.Start = start
	if endStr == "" {
		br.End = ^uint64(0) // max uint64 to indicate open-ended
	} else {
		end, err := strconv.ParseUint(endStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid end block: %w", err)
		}
		if end < start {
			return nil, errors.New("end block must be >= start block")
		}
		br.End = end
	}
	return &br, nil
}

// Validate 检查 CLIConfig 的必需/一致性输入。
func (c *CLIConfig) Validate() error {
	// 如果是下载模式，仅需要下载相关配置
	if c.Download {
		return nil
	}

	if c.AIProvider == "" {
		return errors.New("-ai is required (e.g. -ai chatgpt5)")
	}
	if c.Mode == "" {
		return errors.New("-m (mode) is required: mode1|mode2|mode3")
	}
	if c.Mode != "mode1" && c.Mode != "mode2" && c.Mode != "mode3" {
		return errors.New("-m must be one of: mode1, mode2, mode3")
	}
	// 允许 db | file | contract | address
	if c.TargetSource != "db" && c.TargetSource != "file" && c.TargetSource != "contract" && c.TargetSource != "address" {
		return errors.New("-t must be one of: db, file, contract, address")
	}
	if c.TargetSource == "file" && c.TargetFile == "" {
		return errors.New("-t-file is required when -t=file")
	}
	if (c.TargetSource == "contract" || c.TargetSource == "address") && c.TargetAddress == "" {
		return errors.New("-t-address is required when -t=contract or -t=address")
	}
	if c.Chain == "" {
		c.Chain = "eth" // default
	}
	if c.Concurrency <= 0 {
		c.Concurrency = 4
	}
	return nil
}

// ParseFlags 解析 os.Args 并返回 CLIConfig 或错误。用于从 main 调用。
func ParseFlags() (*CLIConfig, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintln(w, "用法: excavator -ai <provider> -m <mode> [选项]")
		fmt.Fprintln(w)
		fmt.Fprintln(w, "选项:")
		fs.PrintDefaults()
		fmt.Fprintln(w)
		fmt.Fprintln(w, "示例:")
		fmt.Fprintln(w, "  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t file -t-file ./data/source_contracts/sample.yaml -c eth")
		fmt.Fprintln(w, "  excavator -ai chatgpt5 -m mode1 -s all -t db -t-block 1-220234 -c eth")
		fmt.Fprintln(w, "  excavator -d                    # 从上次继续下载")
		fmt.Fprintln(w, "  excavator -d -d-range 1000-2000 # 下载指定区块范围")
	}

	// 新增下载相关 flags（不包含 rpc/dbdsn）
	downloadFlag := fs.Bool("d", false, "启动区块/合约下载流程（从数据库记录的最后区块继续，或使用 -d-range 指定范围）")
	drange := fs.String("d-range", "", "下载区块范围（format start-end），与 -d 一起使用时覆盖从上次继续的行为")
	proxy := fs.String("proxy", "", "可选 HTTP 代理，例如 http://127.0.0.1:7897（下载/请求 Etherscan 时生效）")

	ai := fs.String("ai", "", "AI provider to use (e.g. chatgpt5)")
	mode := fs.String("m", "", "Mode to run: mode1(targeted) | mode2(fuzzy) | mode3(general)")
	strategy := fs.String("s", "all", "Strategy/prompt name in strategy/prompts/<mode>/ (or 'all')")
	target := fs.String("t", "db", "Target source: 'db' or 'file' (default db)")
	blockRange := fs.String("t-block", "", "Block range for scanning (format start-end, e.g. 1-220234)")
	tfile := fs.String("-t-file", "", "YAML file path when -t=file; can be a directory for batching")
	taddress := fs.String("t-address", "", "单个合约地址，当 -t=contract 或 -t=address 时使用")
	chain := fs.String("c", "eth", "Chain to scan: eth | bsc | arb (default eth)")
	concurrency := fs.Int("concurrency", 4, "Worker concurrency")
	verbose := fs.Bool("v", false, "Verbose output")
	timeout := fs.Duration("timeout", 30*time.Second, "Per-AI request timeout")
	fileFlag := fs.String("file", "", "当 -d 一起使用时，从指定 txt 文件读取地址逐条重新下载（每行一个地址）")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	cfg := &CLIConfig{
		AIProvider:    strings.TrimSpace(*ai),
		Mode:          strings.TrimSpace(*mode),
		Strategy:      strings.TrimSpace(*strategy),
		TargetSource:  strings.TrimSpace(*target),
		TargetFile:    strings.TrimSpace(*tfile),
		TargetAddress: strings.TrimSpace(*taddress),
		Chain:         strings.TrimSpace(*chain),
		Concurrency:   *concurrency,
		Verbose:       *verbose,
		Timeout:       *timeout,
		Download:      *downloadFlag,
		Proxy:         strings.TrimSpace(*proxy),
		DownloadFile:  strings.TrimSpace(*fileFlag),
	}

	// 解析下载区块范围（如果提供）
	if strings.TrimSpace(*drange) != "" {
		br, err := parseBlockRange(*drange)
		if err != nil {
			return nil, err
		}
		cfg.DownloadRange = br
	}

	if strings.TrimSpace(*blockRange) != "" {
		br, err := parseBlockRange(*blockRange)
		if err != nil {
			return nil, err
		}
		cfg.BlockRange = br
	}

	// normalize target source
	cfg.TargetSource = strings.ToLower(cfg.TargetSource)
	if cfg.TargetSource == "yaml" {
		cfg.TargetSource = "file" // accept yaml alias
	}

	// 如果提供了文件路径但不是绝对路径，则将其转为相对于当前工作目录
	if cfg.TargetFile != "" {
		if !filepath.IsAbs(cfg.TargetFile) {
			cwd, _ := os.Getwd()
			cfg.TargetFile = filepath.Join(cwd, cfg.TargetFile)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Run 是一个便利包装，解析 flags 并分派到相应处理器。
// 用你实际的内部/核心逻辑替换占位处理调用。
func Run() error {
	cfg, err := ParseFlags()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	// 下载模式优先
	if cfg.Download {
		fmt.Println("🚀 启动合约下载器...")

		// 初始化 MySQL 数据库连接（使用 config.InitDB，不需要传 DSN）
		fmt.Println("📊 正在连接 MySQL 数据库...")
		db, err := config.InitDB()
		if err != nil {
			return fmt.Errorf("初始化数据库失败: %w", err)
		}
		defer db.Close()
		fmt.Println("✅ 数据库连接成功!")

		// 创建下载器（会自动从 config.GetRPCURL() 读取 RPC URL），传入 proxy
		fmt.Println("🔗 正在创建下载器...")
		dl, err := download.NewDownloader(db, cfg.Proxy)
		if err != nil {
			return fmt.Errorf("创建下载器失败: %w", err)
		}
		defer dl.Close()

		// 创建上下文（使用较长的超时时间用于下载）
		ctx := context.Background()

		fmt.Println("\n" + strings.Repeat("=", 50))
		fmt.Println("开始同步合约数据...")
		fmt.Println(strings.Repeat("=", 50) + "\n")

		// 如果提供了下载范围，使用 DownloadBlockRange，否则使用 DownloadFromLast
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

		// 如果用户传入 -file，则从该文件读取地址并逐条重试下载
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
				if line == "" {
					continue
				}
				addrs = append(addrs, line)
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

			fmt.Println("\n🎉 地址重试下载完成!")
			return nil
		}

		// 否则按区块范围/从上次继续下载（原有逻辑）
		fmt.Println("\n🎉 下载任务完成!")
		return nil
	}

	// 非下载模式：正常的扫描流程
	if cfg.Verbose {
		fmt.Printf("使用配置运行 Excavator: %+v\n", cfg)
	}

	// 加载配置文件
	if err := config.LoadSettings("config/settings.yaml"); err != nil {
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
	}
	if cfg.BlockRange != nil {
		internalCfg.BlockRange = &internal.BlockRange{
			Start: cfg.BlockRange.Start,
			End:   cfg.BlockRange.End,
		}
	}

	// TODO: 与内部/核心处理器集成。下面为示例分派。
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
		return errors.New("unsupported mode")
	}

	return nil
}

// PrintFatal 将错误打印到 stderr 并以非零代码退出。
func PrintFatal(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
