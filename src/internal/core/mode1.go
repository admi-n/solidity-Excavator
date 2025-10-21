package core

import (
	"fmt"

	"github.com/admi-n/solidity-Excavator/src/internal"
)

// Mode1Scan 扫描合约并返回原始 AI 输出（占位实现）
func Mode1Scan(contract internal.Contract, prompt string, expCode string, cfg internal.ScanConfig) (string, error) {
	// TODO: 实际扫描逻辑，例如调用 AI 或分析 EXP
	return fmt.Sprintf("Mock AI 输出 for %s", contract.Address), nil
}
