package report

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Storage 报告存储接口
type Storage interface {
	Save(report *Report, content string) (string, error)
}

// FileStorage 文件存储实现
type FileStorage struct {
	OutputDir string
}

// NewFileStorage 创建文件存储
func NewFileStorage(outputDir string) *FileStorage {
	return &FileStorage{
		OutputDir: outputDir,
	}
}

// Save 保存报告到文件
func (s *FileStorage) Save(report *Report, content string) (string, error) {
	// 确保输出目录存在
	if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成文件名
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("scan_report_%s_%d.md", report.Mode, timestamp)
	filepath := filepath.Join(s.OutputDir, filename)

	// 写入文件
	if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write report file: %w", err)
	}

	return filepath, nil
}
