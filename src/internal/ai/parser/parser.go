package parser

// Parser 全局占位对象
var Parser = &aiParser{}

type aiParser struct{}

// Parse 将原始 AI 输出转换为结构化 ScanResult
func (p *aiParser) Parse(raw string) ([]ScanResult, error) {
	// TODO: 解析 AI 输出
	return []ScanResult{
		{
			ContractAddress: "0xDEADBEEF",
			Vulnerability:   "MockVul",
			Severity:        "High",
			Details:         raw,
			Line:            1,
		},
	}, nil
}
