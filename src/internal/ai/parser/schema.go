package parser

type ScanResult struct {
	ContractAddress string
	Vulnerability   string
	Severity        string
	Details         string
	Line            int
}
