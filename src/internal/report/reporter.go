package report

import (
	"fmt"
	"time"
)

// Reporter 报告器，整合生成器和存储功能
type Reporter struct {
	generator Generator
	storage   Storage
}

// NewReporter 创建报告器
func NewReporter(generator Generator, storage Storage) *Reporter {
	return &Reporter{
		generator: generator,
		storage:   storage,
	}
}

// GenerateAndSave 生成并保存报告
func (r *Reporter) GenerateAndSave(report *Report) (string, error) {
	// 生成报告内容
	content, err := r.generator.Generate(report)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}

	// 保存报告
	filepath, err := r.storage.Save(report, content)
	if err != nil {
		return "", fmt.Errorf("failed to save report: %w", err)
	}

	return filepath, nil
}

// NewReport 创建新的报告实例
func NewReport(mode, strategy, aiProvider string) *Report {
	return &Report{
		Mode:                 mode,
		Strategy:             strategy,
		AIProvider:           aiProvider,
		ScanTime:             time.Now(),
		TotalContracts:       0,
		VulnerableContracts:  0,
		SeverityDistribution: make(map[string]int),
		Results:              make([]ScanResult, 0),
	}
}

// AddScanResult 添加扫描结果
func (r *Report) AddScanResult(result ScanResult) {
	r.Results = append(r.Results, result)
	r.TotalContracts++

	if len(result.Vulnerabilities) > 0 {
		r.VulnerableContracts++

		// 统计严重性分布
		for _, vuln := range result.Vulnerabilities {
			r.SeverityDistribution[vuln.Severity]++
		}
	}
}

// NewScanResult 创建新的扫描结果
func NewScanResult(contractAddress string) ScanResult {
	return ScanResult{
		ContractAddress: contractAddress,
		ScanTime:        time.Now(),
		Status:          "✅ 扫描完成",
		Vulnerabilities: make([]Vulnerability, 0),
	}
}

// AddVulnerability 添加漏洞
func (s *ScanResult) AddVulnerability(vuln Vulnerability) {
	s.Vulnerabilities = append(s.Vulnerabilities, vuln)
}

// SetStatus 设置扫描状态
func (s *ScanResult) SetStatus(status string) {
	s.Status = status
}

// SetAnalysisSummary 设置分析摘要
func (s *ScanResult) SetAnalysisSummary(summary string) {
	s.AnalysisSummary = summary
}

// SetRawResponse 设置原始响应
func (s *ScanResult) SetRawResponse(response string) {
	s.RawResponse = response
}
