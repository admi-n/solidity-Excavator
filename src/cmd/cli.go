package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// showHelp 显示帮助信息
func showHelp(topic string) {
	switch topic {
	case "d", "download":
		showDownloadHelp()
	case "ai":
		showAIHelp()
	case "m", "mode":
		showModeHelp()
	case "s", "strategy":
		showStrategyHelp()
	case "t", "target":
		showTargetHelp()
	case "c", "chain":
		showChainHelp()
	default:
		showGeneralHelp()
	}
}

// showGeneralHelp 显示通用帮助
func showGeneralHelp() {
	fmt.Println("🔍 Solidity Excavator - 智能合约安全扫描工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator [命令] [选项]")
	fmt.Println()
	fmt.Println("主要命令:")
	fmt.Println("  -d, --download    启动合约下载模式")
	fmt.Println("  -ai <provider>    指定AI提供商进行扫描")
	fmt.Println("  -m <mode>         指定扫描模式")
	fmt.Println("  -s <strategy>     指定扫描策略")
	fmt.Println("  -t <target>       指定扫描目标")
	fmt.Println("  -c <chain>        指定区块链网络")
	fmt.Println()
	fmt.Println("获取特定命令的帮助:")
	fmt.Println("  excavator -d --help     # 下载模式帮助")
	fmt.Println("  excavator -ai --help    # AI提供商帮助")
	fmt.Println("  excavator -m --help     # 扫描模式帮助")
	fmt.Println("  excavator -s --help     # 扫描策略帮助")
	fmt.Println("  excavator -t --help     # 扫描目标帮助")
	fmt.Println("  excavator -c --help     # 区块链网络帮助")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x123... -c eth")
	fmt.Println("  excavator -d -d-range 1000-2000")
}

// showDownloadHelp 显示下载模式帮助
func showDownloadHelp() {
	fmt.Println("📥 下载模式 (-d, --download)")
	fmt.Println()
	fmt.Println("功能: 从区块链下载合约代码到数据库")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator -d [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -d-range <range>    指定下载区块范围 (格式: start-end)")
	fmt.Println("  -file <path>        从文件读取合约地址进行下载 (独立模式)")
	fmt.Println("  -proxy <url>        使用HTTP代理")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -d                           # 从上次位置继续下载")
	fmt.Println("  excavator -d -d-range 1000-2000        # 下载区块1000-2000")
	fmt.Println("  excavator -d -file contracts.txt      # 只下载文件中的合约地址")
	fmt.Println("  excavator -d -file failed.txt -proxy http://127.0.0.1:7897")
}

// showAIHelp 显示AI提供商帮助
func showAIHelp() {
	fmt.Println("🤖 AI提供商 (-ai)")
	fmt.Println()
	fmt.Println("功能: 指定用于合约分析的AI模型")
	fmt.Println()
	fmt.Println("支持的提供商:")
	fmt.Println("  chatgpt5     OpenAI ChatGPT-5 (推荐)")
	fmt.Println("  openai       OpenAI GPT-4")
	fmt.Println("  gpt4         OpenAI GPT-4")
	fmt.Println("  deepseek     DeepSeek AI")
	fmt.Println("  local-llm    本地LLM (Ollama)")
	fmt.Println("  ollama       本地Ollama")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator -ai <provider> [其他选项]")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x123...")
	fmt.Println("  excavator -ai deepseek -m mode1 -s hourglass-vul -t db -t-block 1-1000")
	fmt.Println("  excavator -ai local-llm -m mode1 -s hourglass-vul -t file -t-file contracts.txt")
	fmt.Println()
	fmt.Println("配置:")
	fmt.Println("  在 config/settings.yaml 中设置API密钥")
	fmt.Println("  或使用环境变量: OPENAI_API_KEY, DEEPSEEK_API_KEY")
}

// showModeHelp 显示扫描模式帮助
func showModeHelp() {
	fmt.Println("🎯 扫描模式 (-m, --mode)")
	fmt.Println()
	fmt.Println("功能: 指定漏洞扫描的模式")
	fmt.Println()
	fmt.Println("支持的模式:")
	fmt.Println("  mode1        定向扫描 - 基于已知漏洞模式进行精确扫描")
	fmt.Println("  mode2        模糊扫描 - 基于相似性进行模糊匹配扫描")
	fmt.Println("  mode3        通用扫描 - 基于SWC标准进行全面扫描")
	fmt.Println()
	fmt.Println("模式详情:")
	fmt.Println("  mode1: 针对特定已知漏洞，使用专门的提示词和EXP代码")
	fmt.Println("  mode2: 基于漏洞特征描述进行相似性匹配")
	fmt.Println("  mode3: 基于SWC和常见漏洞模式进行全面审计")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator -ai <provider> -m <mode> [其他选项]")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x123...")
	fmt.Println("  excavator -ai deepseek -m mode2 -s reentrancy -t db -t-block 1-1000")
	fmt.Println("  excavator -ai chatgpt5 -m mode3 -s all -t file -t-file contracts.txt")
}

// showStrategyHelp 显示扫描策略帮助
func showStrategyHelp() {
	fmt.Println("📋 扫描策略 (-s, --strategy)")
	fmt.Println()
	fmt.Println("功能: 指定具体的扫描策略和提示词")
	fmt.Println()
	fmt.Println("策略类型:")
	fmt.Println("  all          使用所有可用策略")
	fmt.Println("  eg: hourglass-vul")
	fmt.Println()
	fmt.Println("策略文件位置:")
	fmt.Println("  strategy/prompts/mode1/<strategy>.tmpl #提示词")
	fmt.Println("  strategy/exp_libs/mode1/<strategy>.t.sol #漏洞代码/复现")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator -ai <provider> -m <mode> -s <strategy> [其他选项]")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x123...")
	fmt.Println("  excavator -ai deepseek -m mode1 -s all -t db -t-block 1-1000")
	fmt.Println("  excavator -ai chatgpt5 -m mode2 -s reentrancy -t file -t-file contracts.txt")
}

// showTargetHelp 显示扫描目标帮助
func showTargetHelp() {
	fmt.Println("🎯 扫描目标 (-t, --target)")
	fmt.Println()
	fmt.Println("功能: 指定要扫描的合约来源")
	fmt.Println()
	fmt.Println("目标类型:")
	fmt.Println("  contract     扫描单个合约")
	fmt.Println("  address      扫描单个地址 (同contract)")
	fmt.Println("  db           扫描数据库中的合约")
	fmt.Println("  file         扫描文件中的合约地址")
	fmt.Println()
	fmt.Println("相关选项:")
	fmt.Println("  -t-address <addr>    单个合约地址 (与-t contract/address一起使用)")
	fmt.Println("  -t-file <path>        合约地址文件路径 (与-t file一起使用)")
	fmt.Println("  -t-block <range>      区块范围 (与-t db一起使用)")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator -ai <provider> -m <mode> -s <strategy> -t <target> [目标选项]")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x123...")
	fmt.Println("  excavator -ai deepseek -m mode1 -s hourglass-vul -t db -t-block 1-1000")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t file -t-file contracts.txt")
}

// showChainHelp 显示区块链网络帮助
func showChainHelp() {
	fmt.Println("⛓️  区块链网络 (-c, --chain)")
	fmt.Println()
	fmt.Println("功能: 指定要扫描的区块链网络")
	fmt.Println()
	fmt.Println("支持的网络:")
	fmt.Println("  eth         以太坊主网 (默认)")
	fmt.Println("  bsc         Binance Smart Chain")
	fmt.Println("  arb         Arbitrum")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  excavator -ai <provider> -m <mode> -s <strategy> -t <target> -c <chain>")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x123... -c eth")
	fmt.Println("  excavator -ai deepseek -m mode1 -s hourglass-vul -t db -t-block 1-1000 -c bsc")
	fmt.Println("  excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t file -t-file contracts.txt -c arb")
}

// ParseFlags 解析 os.Args 并返回 CLIConfig 或错误。用于从 main 调用。
func ParseFlags() (*CLIConfig, error) {
	// 检查是否请求帮助
	if len(os.Args) > 1 {
		// 处理特定命令的帮助请求 (如 -d --help, -ai --help)
		for i := 1; i < len(os.Args)-1; i++ {
			if os.Args[i+1] == "--help" || os.Args[i+1] == "-h" {
				// 移除前缀的 - 或 --
				cmd := os.Args[i]
				if strings.HasPrefix(cmd, "--") {
					cmd = cmd[2:]
				} else if strings.HasPrefix(cmd, "-") {
					cmd = cmd[1:]
				}
				showHelp(cmd)
				os.Exit(0)
			}
		}

		// 处理通用帮助请求
		for _, arg := range os.Args[1:] {
			if arg == "--help" || arg == "-h" {
				showGeneralHelp()
				os.Exit(0)
			}
		}
	}

	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.Usage = func() {
		showGeneralHelp()
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
	tfile := fs.String("t-file", "", "YAML file path when -t=file; can be a directory for batching")
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
func Run() error {
	cfg, err := ParseFlags()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	return Execute(cfg)
}

// PrintFatal 将错误打印到 stderr 并以非零代码退出。
func PrintFatal(err error) {
	if err == nil {
		return
	}

	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
