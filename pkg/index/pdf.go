package index

import (
	"context"
	"fmt"

	"github.com/JSLEEKR/pageindex-go/pkg/config"
	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// PDFPipeline orchestrates the full PDF-to-tree indexing process.
type PDFPipeline struct {
	client llm.Client
	cfg    *config.Config
}

// NewPDFPipeline creates a new PDF indexing pipeline.
func NewPDFPipeline(client llm.Client, cfg *config.Config) *PDFPipeline {
	return &PDFPipeline{
		client: client,
		cfg:    cfg,
	}
}

// PDFMode represents the processing mode selected for a PDF.
type PDFMode int

const (
	// ModeTOCWithPages - Document has TOC with page numbers
	ModeTOCWithPages PDFMode = iota
	// ModeTOCNoPages - Document has TOC but without page numbers
	ModeTOCNoPages
	// ModeNoTOC - Document has no TOC
	ModeNoTOC
)

// String returns the mode name.
func (m PDFMode) String() string {
	switch m {
	case ModeTOCWithPages:
		return "TOC+pages"
	case ModeTOCNoPages:
		return "TOC-no-pages"
	case ModeNoTOC:
		return "no-TOC"
	default:
		return "unknown"
	}
}

// PDFResult holds the outcome of PDF indexing.
type PDFResult struct {
	Document *tree.Document
	Mode     PDFMode
}

// Index runs the full PDF indexing pipeline.
func (p *PDFPipeline) Index(ctx context.Context, name string, pageTexts []string) (*PDFResult, error) {
	if len(pageTexts) == 0 {
		return nil, fmt.Errorf("no pages to index")
	}

	// Step 1: Try to detect TOC
	detector := NewTOCDetector(p.client, p.cfg.Model, p.cfg.TOCCheckPageNum)
	tocPages, err := detector.FindTOCPages(ctx, pageTexts)
	if err != nil {
		return nil, fmt.Errorf("detecting TOC: %w", err)
	}

	var entries []tree.TOCEntry
	var mode PDFMode

	if len(tocPages) > 0 {
		// Extract TOC
		var tocTexts []string
		for _, pageIdx := range tocPages {
			if pageIdx >= 1 && pageIdx <= len(pageTexts) {
				tocTexts = append(tocTexts, pageTexts[pageIdx-1])
			}
		}

		extractor := NewTOCExtractor(p.client, p.cfg.Model)
		tocText, err := extractor.Extract(ctx, tocTexts)
		if err != nil {
			return nil, fmt.Errorf("extracting TOC: %w", err)
		}

		// Transform to structured entries
		transformer := NewTOCTransformer(p.client, p.cfg.Model)
		entries, err = transformer.Transform(ctx, tocText)
		if err != nil {
			return nil, fmt.Errorf("transforming TOC: %w", err)
		}

		if HasPageNumbers(entries) {
			// Mode A: TOC with page numbers
			mode = ModeTOCWithPages

			// Map to physical pages
			mapper := NewTOCIndexMapper(p.client, p.cfg.Model)
			mappedEntries, err := mapper.MapToPhysicalPages(ctx, entries, pageTexts)
			if err == nil {
				entries = mappedEntries
			}

			// Calculate and apply offset
			offset := CalculatePageOffset(entries)
			entries = ApplyPageOffset(entries, offset)
		} else {
			// Mode B: TOC without page numbers
			mode = ModeTOCNoPages

			// Map entries to physical pages using content
			mapper := NewTOCIndexMapper(p.client, p.cfg.Model)
			mappedEntries, err := mapper.MapToPhysicalPages(ctx, entries, pageTexts)
			if err == nil {
				entries = mappedEntries
			}
		}
	} else {
		// Mode C: No TOC found
		mode = ModeNoTOC

		gen := NewNoTOCGenerator(p.client, p.cfg.Model, p.cfg.TokenLimitPerBatch)
		entries, err = gen.Generate(ctx, pageTexts, 1, len(pageTexts))
		if err != nil {
			return nil, fmt.Errorf("generating TOC: %w", err)
		}
	}

	// Clamp physical indices
	entries = ClampPhysicalIndices(entries, len(pageTexts))

	// Verify and fix
	verifier := NewVerifier(p.client, p.cfg.Model, 5)
	result, err := verifier.Verify(ctx, entries, pageTexts)
	if err == nil && result.Accuracy < 1.0 && result.Accuracy > p.cfg.AccuracyThreshold {
		fixer := NewFixer(p.client, p.cfg.Model, p.cfg.MaxVerifyRetries)
		for retry := 0; retry < p.cfg.MaxVerifyRetries; retry++ {
			entries, err = fixer.FixEntries(ctx, entries, result.IncorrectEntries, pageTexts)
			if err != nil {
				break
			}
			result, err = verifier.Verify(ctx, entries, pageTexts)
			if err != nil || len(result.IncorrectEntries) == 0 {
				break
			}
		}
	}

	// Build tree from flat entries
	nodes := tree.ListToTree(entries)
	tree.SetEndIndices(nodes, len(pageTexts))

	// Process large nodes
	if err := ProcessLargeNodes(ctx, nodes, pageTexts,
		p.cfg.MaxPageNumEachNode, p.cfg.MaxTokenNumEachNode, p.client, p.cfg.Model); err != nil {
		// Non-fatal: continue with existing tree
		_ = err
	}

	// Enrichment
	if p.cfg.AddNodeID {
		tree.AssignNodeIDs(nodes)
	}

	if p.cfg.AddNodeText {
		AttachText(nodes, pageTexts)
	}

	if p.cfg.AddNodeSummary {
		pp := NewPostProcessor(p.client, p.cfg.Model)
		// Attach text temporarily for summary generation
		AttachText(nodes, pageTexts)
		_ = pp.GenerateAllSummaries(ctx, nodes)
		if !p.cfg.AddNodeText {
			// Strip text if not requested
			clearText(nodes)
		}
	}

	// Build document
	doc := &tree.Document{
		DocName:   name,
		Structure: nodes,
		PageCount: len(pageTexts),
	}

	// Build pages array
	for i, text := range pageTexts {
		doc.Pages = append(doc.Pages, tree.Page{PageNum: i + 1, Content: text})
	}

	if p.cfg.AddDocDescription {
		pp := NewPostProcessor(p.client, p.cfg.Model)
		desc, err := pp.GenerateDocDescription(ctx, nodes)
		if err == nil {
			doc.DocDescription = desc
		}
	}

	return &PDFResult{
		Document: doc,
		Mode:     mode,
	}, nil
}

func clearText(nodes []*tree.Node) {
	for _, n := range nodes {
		n.Text = ""
		clearText(n.Children)
	}
}
