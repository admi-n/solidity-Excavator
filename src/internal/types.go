package internal

import "time"

type ScanConfig struct {
	AIProvider   string
	Mode         string
	Strategy     string
	TargetSource string
	TargetFile   string
	Chain        string
	Concurrency  int
	Verbose      bool
	Timeout      time.Duration
	BlockRange   *BlockRange
}

type BlockRange struct {
	Start uint64
	End   uint64
}

// 新增：表示待扫描的合约基础信息
type Contract struct {
	Address string
	Code    string
}
