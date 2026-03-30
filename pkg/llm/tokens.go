package llm

import "strconv"

// EstimateTokens provides a rough token count estimation.
// For OpenAI models, ~4 characters per token on average.
// This is used when an exact tokenizer is not available.
func EstimateTokens(text string) int {
	n := len([]rune(text))
	if n == 0 {
		return 0
	}
	// Rough estimation: ~4 chars per token for English text
	// This is a conservative estimate (slightly over-counts)
	return (n + 3) / 4
}

// BatchByTokenLimit splits texts into batches that fit within a token limit.
// Each text is wrapped with physical_index tags if wrapWithTags is true.
func BatchByTokenLimit(texts []string, limit int, wrapWithTags bool) [][]IndexedText {
	if len(texts) == 0 {
		return nil
	}

	var batches [][]IndexedText
	var currentBatch []IndexedText
	currentTokens := 0

	for i, text := range texts {
		pageIdx := i + 1 // 1-indexed
		wrapped := text
		if wrapWithTags {
			wrapped = WrapWithPhysicalIndex(text, pageIdx)
		}

		tokens := EstimateTokens(wrapped)

		if currentTokens+tokens > limit && len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
			currentBatch = nil
			currentTokens = 0
		}

		currentBatch = append(currentBatch, IndexedText{
			Index:   pageIdx,
			Content: wrapped,
			Raw:     text,
		})
		currentTokens += tokens
	}

	if len(currentBatch) > 0 {
		batches = append(batches, currentBatch)
	}

	return batches
}

// IndexedText holds text with its physical page/line index.
type IndexedText struct {
	Index   int
	Content string // possibly wrapped with tags
	Raw     string // original text
}

// WrapWithPhysicalIndex wraps text with <physical_index_N> tags.
func WrapWithPhysicalIndex(text string, index int) string {
	return "<physical_index_" + itoa(index) + ">\n" + text + "\n</physical_index_" + itoa(index) + ">"
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
