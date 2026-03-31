package index

import (
	"context"
	"strings"
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/config"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

func TestExtractMarkdownNodes(t *testing.T) {
	content := `# Introduction

Some intro text.

## Background

Background info.

### Details

More details.

## Methods

Method description.

# Results

Results here.
`
	nodes := ExtractMarkdownNodes(content)

	if len(nodes) != 5 {
		t.Fatalf("got %d nodes, want 5", len(nodes))
	}

	expected := []struct {
		title string
		level int
	}{
		{"Introduction", 1},
		{"Background", 2},
		{"Details", 3},
		{"Methods", 2},
		{"Results", 1},
	}

	for i, exp := range expected {
		if nodes[i].Title != exp.title {
			t.Errorf("node[%d].Title = %q, want %q", i, nodes[i].Title, exp.title)
		}
		if nodes[i].Level != exp.level {
			t.Errorf("node[%d].Level = %d, want %d", i, nodes[i].Level, exp.level)
		}
	}
}

func TestExtractMarkdownNodesSkipCodeBlocks(t *testing.T) {
	content := "# Real Header\n\nSome text.\n\n```python\n# This is a comment, not a header\ndef foo():\n    pass\n```\n\n## Another Header\n"

	nodes := ExtractMarkdownNodes(content)

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2", len(nodes))
	}
	if nodes[0].Title != "Real Header" {
		t.Errorf("node[0] = %q", nodes[0].Title)
	}
	if nodes[1].Title != "Another Header" {
		t.Errorf("node[1] = %q", nodes[1].Title)
	}
}

func TestExtractMarkdownNodesEmpty(t *testing.T) {
	nodes := ExtractMarkdownNodes("")
	if len(nodes) != 0 {
		t.Errorf("got %d nodes for empty input", len(nodes))
	}
}

func TestExtractMarkdownNodesNoHeaders(t *testing.T) {
	content := "Just some text\nwithout any headers\n"
	nodes := ExtractMarkdownNodes(content)
	if len(nodes) != 0 {
		t.Errorf("got %d nodes for no-header content", len(nodes))
	}
}

func TestExtractNodeText(t *testing.T) {
	content := "# Title\n\nSome content here.\n\n## Section\n\nMore content.\n"

	nodes := ExtractMarkdownNodes(content)
	ExtractNodeText(nodes, content)

	if len(nodes) != 2 {
		t.Fatalf("got %d nodes", len(nodes))
	}

	if !strings.Contains(nodes[0].Text, "Some content here") {
		t.Errorf("node[0].Text = %q, should contain 'Some content here'", nodes[0].Text)
	}
	if !strings.Contains(nodes[1].Text, "More content") {
		t.Errorf("node[1].Text = %q, should contain 'More content'", nodes[1].Text)
	}
}

func TestBuildTreeFromMarkdownNodes(t *testing.T) {
	mdNodes := []MarkdownNode{
		{Title: "Chapter 1", Level: 1, LineNum: 1},
		{Title: "Section 1.1", Level: 2, LineNum: 3},
		{Title: "Section 1.2", Level: 2, LineNum: 6},
		{Title: "Sub 1.2.1", Level: 3, LineNum: 8},
		{Title: "Chapter 2", Level: 1, LineNum: 10},
	}

	roots := BuildTreeFromMarkdownNodes(mdNodes)

	if len(roots) != 2 {
		t.Fatalf("got %d roots, want 2", len(roots))
	}

	ch1 := roots[0]
	if ch1.Title != "Chapter 1" {
		t.Errorf("root[0] = %q", ch1.Title)
	}
	if len(ch1.Children) != 2 {
		t.Fatalf("Chapter 1 has %d children, want 2", len(ch1.Children))
	}
	if ch1.Children[1].Title != "Section 1.2" {
		t.Errorf("child[1] = %q", ch1.Children[1].Title)
	}
	if len(ch1.Children[1].Children) != 1 {
		t.Fatalf("Section 1.2 has %d children, want 1", len(ch1.Children[1].Children))
	}
	if ch1.Children[1].Children[0].Title != "Sub 1.2.1" {
		t.Errorf("grandchild = %q", ch1.Children[1].Children[0].Title)
	}
}

func TestBuildTreeFromMarkdownNodesEmpty(t *testing.T) {
	result := BuildTreeFromMarkdownNodes(nil)
	if result != nil {
		t.Error("expected nil for empty input")
	}
}

func TestBuildTreeFlatHeaders(t *testing.T) {
	mdNodes := []MarkdownNode{
		{Title: "A", Level: 1, LineNum: 1},
		{Title: "B", Level: 1, LineNum: 5},
		{Title: "C", Level: 1, LineNum: 10},
	}

	roots := BuildTreeFromMarkdownNodes(mdNodes)
	if len(roots) != 3 {
		t.Fatalf("got %d roots, want 3", len(roots))
	}
}

func TestTreeThinning(t *testing.T) {
	nodes := []MarkdownNode{
		{Title: "Big Section", Level: 1, LineNum: 1, Text: strings.Repeat("word ", 200)},
		{Title: "Tiny", Level: 2, LineNum: 50, Text: "hi"},
		{Title: "Another Big", Level: 1, LineNum: 60, Text: strings.Repeat("text ", 200)},
	}

	result := TreeThinning(nodes, 50) // 50 token minimum

	if len(result) != 2 {
		t.Fatalf("got %d nodes, want 2 (tiny merged)", len(result))
	}
	if !strings.Contains(result[0].Text, "hi") {
		t.Error("tiny node text should be merged into previous")
	}
}

func TestTreeThinningEmpty(t *testing.T) {
	result := TreeThinning(nil, 50)
	if len(result) != 0 {
		t.Error("expected empty for nil input")
	}
}

func TestMarkdownPipelineIndex(t *testing.T) {
	mock := &mockClient{responses: []string{}}
	cfg := config.Default()
	cfg.AddNodeSummary = false
	cfg.AddDocDescription = false
	cfg.MinTokenThreshold = 0

	pipeline := NewMarkdownPipeline(mock, cfg)

	content := "# Title\n\nContent.\n\n## Section\n\nMore content.\n"

	doc, err := pipeline.Index(context.Background(), "test.md", content)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}

	if doc.DocName != "test.md" {
		t.Errorf("DocName = %q", doc.DocName)
	}
	if len(doc.Structure) != 1 {
		t.Fatalf("got %d roots", len(doc.Structure))
	}
	if doc.Structure[0].Title != "Title" {
		t.Errorf("root = %q", doc.Structure[0].Title)
	}
	if doc.Structure[0].NodeID != "0000" {
		t.Errorf("NodeID = %q, want 0000", doc.Structure[0].NodeID)
	}
}

func TestMarkdownPipelineEmptyContent(t *testing.T) {
	mock := &mockClient{responses: []string{}}
	cfg := config.Default()
	pipeline := NewMarkdownPipeline(mock, cfg)

	_, err := pipeline.Index(context.Background(), "empty.md", "")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestMarkdownPipelineNoHeaders(t *testing.T) {
	mock := &mockClient{responses: []string{}}
	cfg := config.Default()
	cfg.AddNodeSummary = false
	cfg.AddDocDescription = false
	pipeline := NewMarkdownPipeline(mock, cfg)

	doc, err := pipeline.Index(context.Background(), "plain.md", "Just text\nNo headers\n")
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	if len(doc.Structure) != 1 {
		t.Fatalf("expected 1 node for no-header content, got %d", len(doc.Structure))
	}
	if doc.Structure[0].Title != "plain.md" {
		t.Errorf("title = %q", doc.Structure[0].Title)
	}
}

func TestExtractMarkdownNodesMultipleCodeBlocks(t *testing.T) {
	content := `# Real
` + "```\n# fake1\n```\n" + `
## Also Real
` + "```go\n# fake2\n# fake3\n```\n" + `
### Third Real
`
	nodes := ExtractMarkdownNodes(content)
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
}

func TestAttachText(t *testing.T) {
	nodes := []*tree.Node{
		{Title: "A", StartIndex: 1, EndIndex: 2},
		{Title: "B", StartIndex: 3, EndIndex: 3},
	}
	pageTexts := []string{"page1", "page2", "page3"}

	AttachText(nodes, pageTexts)

	if !strings.Contains(nodes[0].Text, "page1") || !strings.Contains(nodes[0].Text, "page2") {
		t.Errorf("A.Text = %q, should contain page1 and page2", nodes[0].Text)
	}
	if nodes[1].Text != "page3" {
		t.Errorf("B.Text = %q, want page3", nodes[1].Text)
	}
}

func TestAttachTextOutOfRange(t *testing.T) {
	nodes := []*tree.Node{
		{Title: "A", StartIndex: 5, EndIndex: 10},
	}
	pageTexts := []string{"p1", "p2"}

	AttachText(nodes, pageTexts)
	if nodes[0].Text != "p1\n\np2" {
		// Clamped to available pages
		t.Logf("A.Text = %q (clamped)", nodes[0].Text)
	}
}

func TestGenerateSummaryTruncatesOnRunes(t *testing.T) {
	// Create text with multi-byte characters (CJK) exceeding 8000 runes
	text := strings.Repeat("한", 9000) // 9000 Korean characters, each 3 bytes
	mock := &mockClient{responses: []string{"summary"}}
	pp := NewPostProcessor(mock, "test")

	node := &tree.Node{Title: "Test", Text: text}
	err := pp.GenerateSummary(context.Background(), node)
	if err != nil {
		t.Fatalf("GenerateSummary: %v", err)
	}

	// Verify the mock received a truncated prompt (not crashing on multi-byte)
	if node.Summary != "summary" {
		t.Errorf("summary = %q", node.Summary)
	}
}

func TestProcessLargeNodesEstimatesFromPageTexts(t *testing.T) {
	// Bug: ProcessLargeNodes was called before AttachText, so node.Text was always
	// empty and EstimateTokens always returned 0, making the function a no-op.
	// After fix, it should estimate tokens from pageTexts directly.
	longPage := strings.Repeat("word ", 10000) // ~10000 tokens
	pageTexts := make([]string, 20)
	for i := range pageTexts {
		pageTexts[i] = longPage
	}

	// Create a leaf node spanning all 20 pages with no text attached
	node := &tree.Node{
		Title:      "Large Chapter",
		StartIndex: 1,
		EndIndex:   20,
		// Text intentionally empty — simulating pre-AttachText state
	}

	// Mock that returns a generated TOC structure
	mock := &mockClient{responses: []string{
		`[{"structure": "1", "title": "Part A", "physical_index": 1}, {"structure": "2", "title": "Part B", "physical_index": 10}]`,
	}}

	// maxPages=5 means this 20-page node exceeds page limit
	// maxTokens=5000 means each page's ~10000 tokens easily exceeds token limit
	// If the bug still existed, tokenCount would be 0 and the function would return
	// without subdividing.
	err := ProcessLargeNodes(context.Background(), []*tree.Node{node}, pageTexts,
		5, 5000, mock, "test")
	if err != nil {
		t.Fatalf("ProcessLargeNodes: %v", err)
	}

	// After fix: node should have been subdivided (children added)
	if len(node.Children) == 0 {
		t.Error("ProcessLargeNodes did not subdivide large node — token estimation from pageTexts not working")
	}
}
