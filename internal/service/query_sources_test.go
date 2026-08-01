package service

import "testing"

func TestShouldShowSources(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		wantShow bool
	}{
		{name: "refusal", answer: "根据现有资料无法回答。", wantShow: false},
		{name: "empty", answer: "  ", wantShow: false},
		{name: "answer", answer: "星期一。", wantShow: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldShowSources(tt.answer); got != tt.wantShow {
				t.Fatalf("ShouldShowSources(%q) = %v, want %v", tt.answer, got, tt.wantShow)
			}
		})
	}
}
