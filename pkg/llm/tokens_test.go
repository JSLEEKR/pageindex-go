package llm

import (
	"strings"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name string
		text string
		min  int
		max  int
	}{
		{"empty", "", 0, 0},
		{"short", "hello", 1, 3},
		{"sentence", "The quick brown fox jumps over the lazy dog", 8, 15},
		{"long", strings.Repeat("word ", 1000), 900, 1500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateTokens(tt.text)
			if got < tt.min || got > tt.max {
				t.Errorf("EstimateTokens(%q...) = %d, want [%d, %d]", tt.text[:min(20, len(tt.text))], got, tt.min, tt.max)
			}
		})
	}
}

func TestBatchByTokenLimit(t *testing.T) {
	texts := []string{
		strings.Repeat("a", 100),
		strings.Repeat("b", 100),
		strings.Repeat("c", 100),
		strings.Repeat("d", 100),
	}

	// Limit high enough for all
	batches := BatchByTokenLimit(texts, 100000, false)
	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %d", len(batches))
	}

	// Limit so each text is its own batch
	batches = BatchByTokenLimit(texts, 30, false)
	if len(batches) != 4 {
		t.Errorf("expected 4 batches, got %d", len(batches))
	}

	// Empty input
	batches = BatchByTokenLimit(nil, 1000, false)
	if batches != nil {
		t.Error("expected nil for empty input")
	}
}

func TestBatchByTokenLimitWithTags(t *testing.T) {
	texts := []string{"page1", "page2"}
	batches := BatchByTokenLimit(texts, 100000, true)

	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if !strings.Contains(batches[0][0].Content, "<physical_index_1>") {
		t.Error("first item should have physical_index_1 tag")
	}
	if !strings.Contains(batches[0][1].Content, "<physical_index_2>") {
		t.Error("second item should have physical_index_2 tag")
	}
	if batches[0][0].Index != 1 {
		t.Errorf("first index = %d, want 1", batches[0][0].Index)
	}
}

func TestWrapWithPhysicalIndex(t *testing.T) {
	result := WrapWithPhysicalIndex("content here", 5)
	expected := "<physical_index_5>\ncontent here\n</physical_index_5>"
	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{42, "42"},
		{100, "100"},
		{-5, "-5"},
	}
	for _, tt := range tests {
		if got := itoa(tt.n); got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

