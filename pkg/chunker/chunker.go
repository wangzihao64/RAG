// Package chunker 把长文本切分成带重叠的小块，供 embedding 使用。
package chunker

import "strings"

// Split 按字符数（rune，兼容中文）把文本切成若干块。
// size 为每块最大字符数，overlap 为相邻块的重叠字符数（用于保留上下文）。
// 返回的每块都会去除首尾空白；空白块会被跳过。
func Split(text string, size, overlap int) []string {
	if size <= 0 {
		size = 500
	}
	// overlap 必须小于 size，否则窗口无法前进
	if overlap < 0 || overlap >= size {
		overlap = size / 5
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	step := size - overlap
	var chunks []string
	// 每隔 step 起一个窗口，直到起点越过文本末尾；
	// 末尾会自然产生一个较短的重叠尾块，保留上下文。
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}
	return chunks
}
