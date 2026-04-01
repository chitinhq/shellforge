package orchestrator

import (
	"strings"
	"testing"
)

func TestCompressResult_ShortOutput(t *testing.T) {
	short := "This is a short output."
	result := CompressResult(short)
	if result != short {
		t.Errorf("short output should pass through unchanged, got %q", result)
	}
}

func TestCompressResult_ExactThreshold(t *testing.T) {
	// 750 tokens * 4 chars = 3000 chars
	exact := strings.Repeat("a", 3000)
	result := CompressResult(exact)
	if result != exact {
		t.Error("output at exact threshold should pass through unchanged")
	}
}

func TestCompressResult_OverThreshold(t *testing.T) {
	// 4000 tokens * 4 chars = 16000 chars
	long := strings.Repeat("x", 16000)
	result := CompressResult(long)

	if len(result) >= len(long) {
		t.Errorf("compressed result should be shorter than original (%d >= %d)", len(result), len(long))
	}
	if !strings.Contains(result, "truncated") {
		t.Error("compressed result should contain truncation marker")
	}
	if !strings.Contains(result, "omitted") {
		t.Error("compressed result should indicate omitted tokens")
	}
}

func TestCompressResult_Empty(t *testing.T) {
	result := CompressResult("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"abcd", 1},
		{"12345678", 2},
		{strings.Repeat("a", 100), 25},
	}
	for _, tt := range tests {
		got := estimateTokens(tt.input)
		if got != tt.expected {
			t.Errorf("estimateTokens(%d chars): expected %d, got %d", len(tt.input), tt.expected, got)
		}
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{-5, "-5"},
		{1000, "1000"},
	}
	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.expected {
			t.Errorf("itoa(%d): expected %q, got %q", tt.input, tt.expected, got)
		}
	}
}
