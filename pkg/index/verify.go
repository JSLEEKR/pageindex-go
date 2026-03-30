package index

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// VerifyResult holds the outcome of TOC verification.
type VerifyResult struct {
	Accuracy         float64
	IncorrectEntries []int // indices of incorrect entries
	TotalChecked     int
}

// Verifier checks if TOC entries correctly map to their claimed pages.
type Verifier struct {
	client     llm.Client
	model      string
	maxWorkers int
}

// NewVerifier creates a new TOC verifier.
func NewVerifier(client llm.Client, model string, maxWorkers int) *Verifier {
	if maxWorkers <= 0 {
		maxWorkers = 5
	}
	return &Verifier{
		client:     client,
		model:      model,
		maxWorkers: maxWorkers,
	}
}

// Verify checks each TOC entry against the actual page content.
func (v *Verifier) Verify(ctx context.Context, entries []tree.TOCEntry, pageTexts []string) (*VerifyResult, error) {
	if len(entries) == 0 {
		return &VerifyResult{Accuracy: 1.0}, nil
	}

	type indexedResult struct {
		idx     int
		correct bool
		err     error
	}

	results := make([]indexedResult, len(entries))
	var mu sync.Mutex
	sem := make(chan struct{}, v.maxWorkers)
	var wg sync.WaitGroup

	for i, entry := range entries {
		if entry.PhysicalIndex < 1 || entry.PhysicalIndex > len(pageTexts) {
			results[i] = indexedResult{idx: i, correct: false}
			continue
		}

		wg.Add(1)
		go func(idx int, e tree.TOCEntry) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				mu.Lock()
				results[idx] = indexedResult{idx: idx, err: ctx.Err()}
				mu.Unlock()
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			pageContent := pageTexts[e.PhysicalIndex-1]
			correct, err := v.verifySingle(ctx, e.Title, e.PhysicalIndex, pageContent)

			mu.Lock()
			results[idx] = indexedResult{idx: idx, correct: correct, err: err}
			mu.Unlock()
		}(i, entry)
	}

	wg.Wait()

	var incorrect []int
	correctCount := 0

	for i, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("verifying entry %d: %w", i, r.err)
		}
		if r.correct {
			correctCount++
		} else {
			incorrect = append(incorrect, i)
		}
	}

	accuracy := float64(correctCount) / float64(len(entries))

	return &VerifyResult{
		Accuracy:         accuracy,
		IncorrectEntries: incorrect,
		TotalChecked:     len(entries),
	}, nil
}

func (v *Verifier) verifySingle(ctx context.Context, title string, physIdx int, pageContent string) (bool, error) {
	prompt := fmt.Sprintf(llm.PromptVerifyTOC, title, physIdx, pageContent)

	resp, err := v.client.Complete(ctx, llm.CompletionRequest{
		Model:       v.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(strings.ToLower(resp)) == "yes", nil
}

// Fixer attempts to correct incorrectly mapped TOC entries.
type Fixer struct {
	client     llm.Client
	model      string
	maxRetries int
}

// NewFixer creates a new TOC fixer.
func NewFixer(client llm.Client, model string, maxRetries int) *Fixer {
	return &Fixer{
		client:     client,
		model:      model,
		maxRetries: maxRetries,
	}
}

// FixEntries attempts to fix incorrectly mapped entries.
func (f *Fixer) FixEntries(ctx context.Context, entries []tree.TOCEntry, incorrectIndices []int, pageTexts []string) ([]tree.TOCEntry, error) {
	fixed := make([]tree.TOCEntry, len(entries))
	copy(fixed, entries)

	for _, idx := range incorrectIndices {
		if idx < 0 || idx >= len(fixed) {
			continue
		}

		entry := fixed[idx]

		// Get surrounding pages
		startPage := entry.PhysicalIndex - 3
		if startPage < 1 {
			startPage = 1
		}
		endPage := entry.PhysicalIndex + 3
		if endPage > len(pageTexts) {
			endPage = len(pageTexts)
		}

		var surroundingPages []string
		for p := startPage; p <= endPage; p++ {
			surroundingPages = append(surroundingPages,
				llm.WrapWithPhysicalIndex(pageTexts[p-1], p))
		}

		pagesText := strings.Join(surroundingPages, "\n\n")
		prompt := fmt.Sprintf(llm.PromptFixTOCEntry, entry.Title, entry.PhysicalIndex, pagesText)

		resp, err := f.client.Complete(ctx, llm.CompletionRequest{
			Model:       f.model,
			Messages:    []llm.Message{llm.UserPrompt(prompt)},
			Temperature: 0,
		})
		if err != nil {
			continue // Skip entries we can't fix
		}

		newPage, err := strconv.Atoi(strings.TrimSpace(resp))
		if err == nil && newPage >= 1 && newPage <= len(pageTexts) {
			fixed[idx].PhysicalIndex = newPage
		}
	}

	return fixed, nil
}
