// Package index implements the PDF and Markdown indexing pipelines.
package index

import (
	"context"
	"fmt"
	"strings"

	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// TOCDetector scans PDF pages to find table of contents pages.
type TOCDetector struct {
	client    llm.Client
	model     string
	maxPages  int
}

// NewTOCDetector creates a new TOC detector.
func NewTOCDetector(client llm.Client, model string, maxPages int) *TOCDetector {
	return &TOCDetector{
		client:   client,
		model:    model,
		maxPages: maxPages,
	}
}

// FindTOCPages scans the first N pages to identify TOC pages.
// Returns the indices (1-based) of pages that are part of the TOC.
func (d *TOCDetector) FindTOCPages(ctx context.Context, pages []string) ([]int, error) {
	limit := d.maxPages
	if limit > len(pages) {
		limit = len(pages)
	}

	var tocPages []int
	foundTOC := false

	for i := 0; i < limit; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		isTOC, err := d.detectSinglePage(ctx, pages[i])
		if err != nil {
			return nil, fmt.Errorf("detecting TOC on page %d: %w", i+1, err)
		}

		if isTOC {
			tocPages = append(tocPages, i+1)
			foundTOC = true
		} else if foundTOC {
			// Stop once we find a non-TOC page after finding TOC pages
			break
		}
	}

	return tocPages, nil
}

func (d *TOCDetector) detectSinglePage(ctx context.Context, pageText string) (bool, error) {
	prompt := fmt.Sprintf(llm.PromptTOCDetector, pageText)

	resp, err := d.client.Complete(ctx, llm.CompletionRequest{
		Model:       d.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(strings.ToLower(resp)) == "yes", nil
}

// TOCExtractor extracts raw TOC text from identified TOC pages.
type TOCExtractor struct {
	client llm.Client
	model  string
}

// NewTOCExtractor creates a new TOC extractor.
func NewTOCExtractor(client llm.Client, model string) *TOCExtractor {
	return &TOCExtractor{client: client, model: model}
}

// Extract extracts raw TOC text from the given pages.
func (e *TOCExtractor) Extract(ctx context.Context, pages []string) (string, error) {
	combined := strings.Join(pages, "\n\n---\n\n")
	prompt := fmt.Sprintf(llm.PromptTOCExtractor, combined)

	resp, err := e.client.Complete(ctx, llm.CompletionRequest{
		Model:       e.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return "", fmt.Errorf("extracting TOC: %w", err)
	}

	return strings.TrimSpace(resp), nil
}

// TOCTransformer converts raw TOC text to structured entries.
type TOCTransformer struct {
	client llm.Client
	model  string
}

// NewTOCTransformer creates a new TOC transformer.
func NewTOCTransformer(client llm.Client, model string) *TOCTransformer {
	return &TOCTransformer{client: client, model: model}
}

// Transform converts raw TOC text to structured TOC entries.
func (t *TOCTransformer) Transform(ctx context.Context, tocText string) ([]tree.TOCEntry, error) {
	prompt := fmt.Sprintf(llm.PromptTOCTransformer, tocText)

	resp, err := t.client.Complete(ctx, llm.CompletionRequest{
		Model:       t.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("transforming TOC: %w", err)
	}

	entries, err := llm.ExtractJSON[[]tree.TOCEntry](resp)
	if err != nil {
		return nil, fmt.Errorf("parsing TOC JSON: %w", err)
	}

	return entries, nil
}

// TOCIndexMapper maps TOC entries to physical PDF page locations.
type TOCIndexMapper struct {
	client llm.Client
	model  string
}

// NewTOCIndexMapper creates a new TOC index mapper.
func NewTOCIndexMapper(client llm.Client, model string) *TOCIndexMapper {
	return &TOCIndexMapper{client: client, model: model}
}

// MapToPhysicalPages maps TOC entries to physical page locations using document content.
func (m *TOCIndexMapper) MapToPhysicalPages(ctx context.Context, entries []tree.TOCEntry, pageTexts []string) ([]tree.TOCEntry, error) {
	// Wrap pages with physical_index tags
	var wrappedPages []string
	for i, text := range pageTexts {
		wrappedPages = append(wrappedPages, llm.WrapWithPhysicalIndex(text, i+1))
	}

	tocJSON, _ := marshalJSON(entries)
	pagesText := strings.Join(wrappedPages, "\n\n")

	prompt := fmt.Sprintf(llm.PromptTOCIndexExtractor, tocJSON, pagesText)

	resp, err := m.client.Complete(ctx, llm.CompletionRequest{
		Model:       m.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("mapping TOC to physical pages: %w", err)
	}

	result, err := llm.ExtractJSON[[]tree.TOCEntry](resp)
	if err != nil {
		return nil, fmt.Errorf("parsing mapped TOC JSON: %w", err)
	}

	return result, nil
}

// CalculatePageOffset computes the offset between TOC page numbers and physical pages.
// Returns offset such that physical_page = toc_page + offset.
func CalculatePageOffset(entries []tree.TOCEntry) int {
	var offsets []int
	for _, e := range entries {
		if e.PageNumber > 0 && e.PhysicalIndex > 0 {
			offsets = append(offsets, e.PhysicalIndex-e.PageNumber)
		}
	}

	if len(offsets) == 0 {
		return 0
	}

	// Return the most common offset (mode)
	counts := make(map[int]int)
	for _, o := range offsets {
		counts[o]++
	}

	bestOffset := 0
	bestCount := 0
	for offset, count := range counts {
		if count > bestCount || (count == bestCount && offset < bestOffset) {
			bestCount = count
			bestOffset = offset
		}
	}

	return bestOffset
}

// ApplyPageOffset adds the offset to all TOC entries' page numbers
// to convert them to physical page indices.
func ApplyPageOffset(entries []tree.TOCEntry, offset int) []tree.TOCEntry {
	result := make([]tree.TOCEntry, len(entries))
	for i, e := range entries {
		result[i] = e
		if e.PhysicalIndex == 0 && e.PageNumber > 0 {
			result[i].PhysicalIndex = e.PageNumber + offset
		}
	}
	return result
}

// HasPageNumbers checks if TOC entries have page numbers.
func HasPageNumbers(entries []tree.TOCEntry) bool {
	for _, e := range entries {
		if e.PageNumber > 0 {
			return true
		}
	}
	return false
}

// ClampPhysicalIndices ensures all physical indices are within valid range.
func ClampPhysicalIndices(entries []tree.TOCEntry, maxPage int) []tree.TOCEntry {
	result := make([]tree.TOCEntry, len(entries))
	for i, e := range entries {
		result[i] = e
		if result[i].PhysicalIndex > maxPage {
			result[i].PhysicalIndex = maxPage
		}
		if result[i].PhysicalIndex < 1 && result[i].PhysicalIndex != 0 {
			result[i].PhysicalIndex = 1
		}
	}
	return result
}
