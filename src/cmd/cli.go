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

// CLIConfig 保存解析好的 CLI 选项以及供扫描器使用的规范化字段。
type CLIConfig struct {
	AIProvider   string // 例如 chatgpt5
	Mode         string // mode1 | mode2 | mode3
	Strategy     string // 例如 hourglass-vul 或 "all"
	TargetSource string // "db" 或 "file" - 从哪里获取目标列表
	BlockRange   *BlockRange
	TargetFile   string // 包含地址/批次的 YAML 路径
	Chain        string // eth | bsc | arb
	Concurrency  int
	Verbose      bool
	Timeout      time.Duration
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
	if c.AIProvider == "" {
		return errors.New("-ai is required (e.g. -ai chatgpt5)")
	}
	if c.Mode == "" {
		return errors.New("-m (mode) is required: mode1|mode2|mode3")
	}
	if c.Mode != "mode1" && c.Mode != "mode2" && c.Mode != "mode3" {
		return errors.New("-m must be one of: mode1, mode2, mode3")
	}
	if c.TargetSource != "db" && c.TargetSource != "file" {
		return errors.New("-t must be either 'db' or 'file'")
	}
	if c.TargetSource == "file" && c.TargetFile == "" {
		return errors.New("-t-file is required when -t=file")
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
	}

	ai := fs.String("ai", "", "AI provider to use (e.g. chatgpt5)")
	mode := fs.String("m", "", "Mode to run: mode1 | mode2 | mode3")
	strategy := fs.String("s", "all", "Strategy/prompt name in strategy/prompts/<mode>/ (or 'all')")
	target := fs.String("t", "db", "Target source: 'db' or 'file' (default db)")
	blockRange := fs.String("t-block", "", "Block range for scanning (format start-end, e.g. 1-220234)")
	tfile := fs.String("t-file", "", "YAML file path when -t=file; can be a directory for batching")
	chain := fs.String("c", "eth", "Chain to scan: eth | bsc | arb (default eth)")
	concurrency := fs.Int("concurrency", 4, "Worker concurrency")
	verbose := fs.Bool("v", false, "Verbose output")
	timeout := fs.Duration("timeout", 30*time.Second, "Per-AI request timeout")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	cfg := &CLIConfig{
		AIProvider:   strings.TrimSpace(*ai),
		Mode:         strings.TrimSpace(*mode),
		Strategy:     strings.TrimSpace(*strategy),
		TargetSource: strings.TrimSpace(*target),
		TargetFile:   strings.TrimSpace(*tfile),
		Chain:        strings.TrimSpace(*chain),
		Concurrency:  *concurrency,
		Verbose:      *verbose,
		Timeout:      *timeout,
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

	if cfg.Verbose {
		fmt.Printf("使用配置运行 Excavator: %+v\n", cfg)
	}

	// TODO: 与内部/核心处理器集成。下面为示例分派。
	switch cfg.Mode {
	case "mode1":
		fmt.Println("分派到 mode1（定向）处理器 — 请实现调用 internal/handler")
	case "mode2":
		fmt.Println("分派到 mode2（模糊）处理器 — 请实现调用 internal/handler")
	case "mode3":
		fmt.Println("分派到 mode3（通用）处理器 — 请实现调用 internal/handler")
	default:
		return errors.New("unsupported mode")
	}

	return nil
}

// 小帮助函数：PrintFatal 将错误打印到 stderr 并以非零代码退出。
func PrintFatal(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "错误:", err)
	os.Exit(1)
}
