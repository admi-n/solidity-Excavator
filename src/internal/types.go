package internal

import "time"

type ScanConfig struct {
	AIProvider    string
	Mode          string
	Strategy      string
	TargetSource  string
	TargetFile    string
	TargetAddress string
	Chain         string
	Concurrency   int
	Verbose       bool
	Timeout       time.Duration
	BlockRange    *BlockRange
}

type BlockRange struct {
	Start uint64
	End   uint64
}

// Contract 表示待扫描的合约基础信息，包含数据库表字段映射
type Contract struct {
	Address      string    `json:"address"`      // 合约地址
	Code         string    `json:"contract"`     // 合约代码
	Balance      string    `json:"balance"`      // 余额（以字符串保存以避免精度/类型问题）
	IsOpenSource bool      `json:"isOpenSource"` // 是否开源 (true/false 对应 1/0)
	CreateTime   time.Time `json:"createtime"`   // 创建时间
	CreateBlock  uint64    `json:"createblock"`  // 创建区块号
	TxLast       time.Time `json:"txlast"`       // 最后一次交互时间
	IsDecompiled bool      `json:"isdecompiled"` //是否开源
	DedCode      string    `json:"dedcode"`      //伪代码

}
