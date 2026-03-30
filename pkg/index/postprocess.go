package index

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// PostProcessor handles tree enrichment operations.
type PostProcessor struct {
	client llm.Client
	model  string
}

// NewPostProcessor creates a new post-processor.
func NewPostProcessor(client llm.Client, model string) *PostProcessor {
	return &PostProcessor{client: client, model: model}
}

// AttachText attaches raw page text to each node based on page ranges.
func AttachText(nodes []*tree.Node, pageTexts []string) {
	for _, n := range nodes {
		if n.StartIndex > 0 && n.EndIndex > 0 {
			var texts []string
			start := n.StartIndex
			end := n.EndIndex
			if start < 1 {
				start = 1
			}
			if end > len(pageTexts) {
				end = len(pageTexts)
			}
			for p := start; p <= end; p++ {
				texts = append(texts, pageTexts[p-1])
			}
			n.Text = strings.Join(texts, "\n\n")
		}
		if len(n.Children) > 0 {
			AttachText(n.Children, pageTexts)
		}
	}
}

// GenerateSummary generates a summary for a single node.
func (pp *PostProcessor) GenerateSummary(ctx context.Context, node *tree.Node) error {
	if node.Text == "" {
		return nil
	}

	// Truncate text if too long (rune-safe for multi-byte characters)
	text := node.Text
	runes := []rune(text)
	if len(runes) > 8000 {
		text = string(runes[:8000]) + "\n[truncated]"
	}

	prompt := fmt.Sprintf(llm.PromptGenerateSummary, node.Title, text)

	resp, err := pp.client.Complete(ctx, llm.CompletionRequest{
		Model:       pp.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return fmt.Errorf("generating summary for %q: %w", node.Title, err)
	}

	node.Summary = strings.TrimSpace(resp)
	return nil
}

// GenerateAllSummaries generates summaries for all nodes concurrently.
func (pp *PostProcessor) GenerateAllSummaries(ctx context.Context, nodes []*tree.Node) error {
	flat := tree.Flatten(nodes)
	const maxWorkers = 5

	sem := make(chan struct{}, maxWorkers)
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup

	for _, n := range flat {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		mu.Lock()
		if firstErr != nil {
			mu.Unlock()
			break
		}
		mu.Unlock()

		wg.Add(1)
		go func(node *tree.Node) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			case sem <- struct{}{}:
				defer func() { <-sem }()
			}

			if err := pp.GenerateSummary(ctx, node); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(n)
	}

	wg.Wait()
	return firstErr
}

// GenerateDocDescription generates a one-sentence document description.
func (pp *PostProcessor) GenerateDocDescription(ctx context.Context, nodes []*tree.Node) (string, error) {
	stripped := tree.StripText(nodes)
	jsonStr, err := marshalJSON(stripped)
	if err != nil {
		return "", fmt.Errorf("marshaling tree: %w", err)
	}

	prompt := fmt.Sprintf(llm.PromptDocDescription, jsonStr)

	resp, err := pp.client.Complete(ctx, llm.CompletionRequest{
		Model:       pp.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return "", fmt.Errorf("generating doc description: %w", err)
	}

	return strings.TrimSpace(resp), nil
}

// ProcessLargeNodes recursively subdivides nodes that exceed limits.
func ProcessLargeNodes(ctx context.Context, nodes []*tree.Node, pageTexts []string,
	maxPages int, maxTokens int, client llm.Client, model string) error {

	for _, n := range nodes {
		if err := processLargeNode(ctx, n, pageTexts, maxPages, maxTokens, client, model); err != nil {
			return err
		}
	}
	return nil
}

func processLargeNode(ctx context.Context, node *tree.Node, pageTexts []string,
	maxPages int, maxTokens int, client llm.Client, model string) error {

	if !node.IsLeaf() {
		// Process children first
		for _, ch := range node.Children {
			if err := processLargeNode(ctx, ch, pageTexts, maxPages, maxTokens, client, model); err != nil {
				return err
			}
		}
		return nil
	}

	pageSpan := node.PageSpan()
	if pageSpan <= maxPages {
		return nil
	}

	tokenCount := llm.EstimateTokens(node.Text)
	if tokenCount <= maxTokens {
		return nil
	}

	// This node is too large — subdivide using no-TOC generation
	gen := NewNoTOCGenerator(client, model, maxTokens)
	subEntries, err := gen.Generate(ctx, pageTexts, node.StartIndex, node.EndIndex)
	if err != nil {
		return fmt.Errorf("subdividing node %q: %w", node.Title, err)
	}

	if len(subEntries) > 0 {
		node.Children = tree.ListToTree(subEntries)
		tree.SetEndIndices(node.Children, node.EndIndex)
	}

	return nil
}

// NoTOCGenerator generates tree structure for documents without a TOC.
type NoTOCGenerator struct {
	client     llm.Client
	model      string
	tokenLimit int
}

// NewNoTOCGenerator creates a new no-TOC generator.
func NewNoTOCGenerator(client llm.Client, model string, tokenLimit int) *NoTOCGenerator {
	return &NoTOCGenerator{
		client:     client,
		model:      model,
		tokenLimit: tokenLimit,
	}
}

// Generate creates a tree structure from pages without a TOC.
func (g *NoTOCGenerator) Generate(ctx context.Context, pageTexts []string, startPage, endPage int) ([]tree.TOCEntry, error) {
	if startPage < 1 {
		startPage = 1
	}
	if endPage > len(pageTexts) {
		endPage = len(pageTexts)
	}

	// Select the relevant page range
	relevantPages := pageTexts[startPage-1 : endPage]

	// Batch pages by token limit
	batches := llm.BatchByTokenLimit(relevantPages, g.tokenLimit, true)
	if len(batches) == 0 {
		return nil, nil
	}

	// Adjust page indices for batches (they start from startPage, not 1)
	for bi := range batches {
		for pi := range batches[bi] {
			batches[bi][pi].Index = startPage + batches[bi][pi].Index - 1
			batches[bi][pi].Content = llm.WrapWithPhysicalIndex(batches[bi][pi].Raw, batches[bi][pi].Index)
		}
	}

	// Generate initial structure from first batch
	var batchTexts []string
	for _, item := range batches[0] {
		batchTexts = append(batchTexts, item.Content)
	}
	pagesText := strings.Join(batchTexts, "\n\n")

	prompt := fmt.Sprintf(llm.PromptGenerateTOCInit, pagesText)
	resp, err := g.client.Complete(ctx, llm.CompletionRequest{
		Model:       g.model,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return nil, fmt.Errorf("generating initial TOC: %w", err)
	}

	allEntries, err := llm.ExtractJSON[[]tree.TOCEntry](resp)
	if err != nil {
		return nil, fmt.Errorf("parsing initial TOC: %w", err)
	}

	// Continue with subsequent batches
	for i := 1; i < len(batches); i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		prevJSON, _ := marshalJSON(allEntries)

		batchTexts = nil
		for _, item := range batches[i] {
			batchTexts = append(batchTexts, item.Content)
		}
		pagesText = strings.Join(batchTexts, "\n\n")

		prompt = fmt.Sprintf(llm.PromptGenerateTOCContinue, prevJSON, pagesText)
		resp, err = g.client.Complete(ctx, llm.CompletionRequest{
			Model:       g.model,
			Messages:    []llm.Message{llm.UserPrompt(prompt)},
			Temperature: 0,
		})
		if err != nil {
			continue // best effort for additional batches
		}

		newEntries, err := llm.ExtractJSON[[]tree.TOCEntry](resp)
		if err != nil {
			continue
		}

		allEntries = append(allEntries, newEntries...)
	}

	return allEntries, nil
}
