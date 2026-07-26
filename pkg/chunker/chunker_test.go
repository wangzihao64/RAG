package chunker

import (
	"strings"
	"testing"
)

func TestSplitEmpty(t *testing.T) {
	if got := Split("", 100, 20); got != nil {
		t.Errorf("空文本应返回 nil，得到 %v", got)
	}
	if got := Split("   \n  ", 100, 20); got != nil {
		t.Errorf("纯空白应返回 nil，得到 %v", got)
	}
}

func TestSplitShorterThanSize(t *testing.T) {
	got := Split("hello world", 100, 20)
	if len(got) != 1 || got[0] != "hello world" {
		t.Errorf("短文本应为单块，得到 %v", got)
	}
}

func TestSplitChineseByRune(t *testing.T) {
	// 10 个汉字，size=4 overlap=1 => step=3，窗口 [0:4][3:7][6:10][9:10]
	text := "零一二三四五六七八九"
	got := Split(text, 4, 1)
	want := []string{"零一二三", "三四五六", "六七八九", "九"}
	if len(got) != len(want) {
		t.Fatalf("块数不符，want %d got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 块 want %q got %q", i, want[i], got[i])
		}
	}
	// 每块都必须是合法 UTF-8（未从中间截断多字节字符）
	for _, c := range got {
		if !strings.ContainsAny(c, "零一二三四五六七八九") {
			t.Errorf("块内容异常: %q", c)
		}
	}
}

func TestSplitOverlap(t *testing.T) {
	// size=5 overlap=2 => step=3
	got := Split("abcdefgh", 5, 2)
	want := []string{"abcde", "defgh", "gh"}
	if len(got) != len(want) {
		t.Fatalf("块数不符，want %v got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 块 want %q got %q", i, want[i], got[i])
		}
	}
}

func TestSplitInvalidOverlapFallback(t *testing.T) {
	// overlap >= size 时应回退，不能死循环
	got := Split("abcdefghij", 5, 10)
	if len(got) == 0 {
		t.Error("非法 overlap 应回退并正常切分")
	}
}
