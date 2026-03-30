package index

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/JSLEEKR/pageindex-go/pkg/config"
	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

var headerRE = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

// MarkdownNode is an intermediate representation during markdown parsing.
type MarkdownNode struct {
	Title   string
	Level   int
	LineNum int
	Text    string
}

// MarkdownPipeline handles markdown-to-tree indexing.
type MarkdownPipeline struct {
	client llm.Client
	cfg    *config.Config
}

// NewMarkdownPipeline creates a new markdown pipeline.
func NewMarkdownPipeline(client llm.Client, cfg *config.Config) *MarkdownPipeline {
	return &MarkdownPipeline{
		client: client,
		cfg:    cfg,
	}
}

// Index processes a markdown document into a tree structure.
func (mp *MarkdownPipeline) Index(ctx context.Context, name string, content string) (*tree.Document, error) {
	if content == "" {
		return nil, fmt.Errorf("empty markdown content")
	}

	// Extract header nodes
	mdNodes := ExtractMarkdownNodes(content)
	if len(mdNodes) == 0 {
		// No headers found — treat entire document as single node
		lines := strings.Split(content, "\n")
		doc := &tree.Document{
			DocName:   name,
			Structure: []*tree.Node{{Title: name, LineNum: 1, Text: content}},
			LineCount: len(lines),
		}
		if mp.cfg.AddNodeID {
			tree.AssignNodeIDs(doc.Structure)
		}
		return doc, nil
	}

	// Extract text content for each node
	ExtractNodeText(mdNodes, content)

	// Optional: tree thinning for small nodes
	if mp.cfg.MinTokenThreshold > 0 {
		mdNodes = TreeThinning(mdNodes, mp.cfg.MinTokenThreshold)
	}

	// Build tree
	nodes := BuildTreeFromMarkdownNodes(mdNodes)

	// Enrichment
	if mp.cfg.AddNodeID {
		tree.AssignNodeIDs(nodes)
	}

	if mp.cfg.AddNodeSummary {
		pp := NewPostProcessor(mp.client, mp.cfg.Model)
		_ = pp.GenerateAllSummaries(ctx, nodes)
	}

	lines := strings.Split(content, "\n")
	doc := &tree.Document{
		DocName:   name,
		Structure: nodes,
		LineCount: len(lines),
	}

	if mp.cfg.AddDocDescription {
		pp := NewPostProcessor(mp.client, mp.cfg.Model)
		desc, err := pp.GenerateDocDescription(ctx, nodes)
		if err == nil {
			doc.DocDescription = desc
		}
	}

	return doc, nil
}

// ExtractMarkdownNodes extracts header nodes from markdown content.
// It skips headers inside code blocks.
func ExtractMarkdownNodes(content string) []MarkdownNode {
	lines := strings.Split(content, "\n")
	var nodes []MarkdownNode
	inCodeBlock := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for code block boundaries
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		matches := headerRE.FindStringSubmatch(trimmed)
		if matches != nil {
			level := len(matches[1]) // number of # characters
			title := strings.TrimSpace(matches[2])
			nodes = append(nodes, MarkdownNode{
				Title:   title,
				Level:   level,
				LineNum: i + 1, // 1-indexed
			})
		}
	}

	return nodes
}

// ExtractNodeText assigns text content to each markdown node.
// Text is the content from this header to the next header.
func ExtractNodeText(nodes []MarkdownNode, content string) {
	lines := strings.Split(content, "\n")

	for i := range nodes {
		startLine := nodes[i].LineNum // 1-indexed, this is the header line
		var endLine int
		if i+1 < len(nodes) {
			endLine = nodes[i+1].LineNum - 1
		} else {
			endLine = len(lines)
		}

		// Collect lines from after the header to the end
		var textLines []string
		for l := startLine; l <= endLine; l++ {
			if l-1 >= 0 && l-1 < len(lines) {
				textLines = append(textLines, lines[l-1])
			}
		}

		nodes[i].Text = strings.TrimSpace(strings.Join(textLines, "\n"))
	}
}

// TreeThinning merges nodes below the minimum token threshold into their parent.
func TreeThinning(nodes []MarkdownNode, minTokens int) []MarkdownNode {
	if len(nodes) == 0 {
		return nodes
	}

	var result []MarkdownNode
	for _, n := range nodes {
		tokens := llm.EstimateTokens(n.Text)
		if tokens >= minTokens || len(result) == 0 {
			result = append(result, n)
		} else {
			// Merge into previous node
			prev := &result[len(result)-1]
			prev.Text = prev.Text + "\n\n" + n.Text
		}
	}

	return result
}

// BuildTreeFromMarkdownNodes converts flat markdown nodes into a nested tree.
// Uses a stack-based approach: pop ancestors until finding a parent at a lower level.
func BuildTreeFromMarkdownNodes(mdNodes []MarkdownNode) []*tree.Node {
	if len(mdNodes) == 0 {
		return nil
	}

	type stackItem struct {
		node  *tree.Node
		level int
	}

	var roots []*tree.Node
	var stack []stackItem

	for _, md := range mdNodes {
		node := &tree.Node{
			Title:   md.Title,
			LineNum: md.LineNum,
			Text:    md.Text,
		}

		// Pop stack until we find a parent at a lower level
		for len(stack) > 0 && stack[len(stack)-1].level >= md.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, node)
		}

		stack = append(stack, stackItem{node: node, level: md.Level})
	}

	return roots
}
