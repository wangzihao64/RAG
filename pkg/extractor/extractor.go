// Package extractor 从各类文档中提取纯文本，供切分与向量化使用。
package extractor

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// ErrUnsupportedType 表示暂不支持解析该类型的文档
var ErrUnsupportedType = errors.New("暂不支持解析该文件类型")

// Extract 按文件类型提取纯文本。fileType 为不带点的小写扩展名，如 txt/md/pdf。
func Extract(path, fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "txt", "md", "markdown":
		return extractPlainText(path)
	case "pdf", "docx":
		// 解析这两类需要额外的库，后续单独实现
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, fileType)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedType, fileType)
	}
}

// extractPlainText 直接读取纯文本文件内容
func extractPlainText(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
