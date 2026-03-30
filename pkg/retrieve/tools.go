// Package retrieve provides tool functions for agentic document retrieval.
package retrieve

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// DocumentInfo is the response from GetDocument.
type DocumentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	PageCount   int    `json:"page_count,omitempty"`
	LineCount   int    `json:"line_count,omitempty"`
	NodeCount   int    `json:"node_count"`
}

// GetDocument returns metadata about a document.
func GetDocument(doc *tree.Document) (*DocumentInfo, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	nodeCount := 0
	for _, n := range doc.Structure {
		nodeCount += 1 + n.TotalDescendants()
	}

	return &DocumentInfo{
		Name:        doc.DocName,
		Description: doc.DocDescription,
		PageCount:   doc.PageCount,
		LineCount:   doc.LineCount,
		NodeCount:   nodeCount,
	}, nil
}

// GetDocumentStructure returns the tree structure without text (to save tokens).
func GetDocumentStructure(doc *tree.Document) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}

	stripped := tree.StripText(doc.Structure)

	data, err := json.MarshalIndent(stripped, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling structure: %w", err)
	}

	return string(data), nil
}

// GetPageContent returns the raw text for specified pages.
// pageSpec can be:
//   - "5" (single page)
//   - "5-7" (range)
//   - "3,5,7" (list)
//   - "1-3,5,7-9" (mixed)
func GetPageContent(doc *tree.Document, pageSpec string) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}

	pages, err := ParsePageSpec(pageSpec)
	if err != nil {
		return "", fmt.Errorf("invalid page spec %q: %w", pageSpec, err)
	}

	var parts []string
	for _, p := range pages {
		content, err := getPageByNumber(doc, p)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("=== Page %d ===\n%s", p, content))
	}

	return strings.Join(parts, "\n\n"), nil
}

// GetNodeContent returns the text content for a specific node.
func GetNodeContent(doc *tree.Document, nodeID string) (string, error) {
	if doc == nil {
		return "", fmt.Errorf("document is nil")
	}

	node := tree.FindByID(doc.Structure, nodeID)
	if node == nil {
		return "", fmt.Errorf("node %q not found", nodeID)
	}

	if node.Text != "" {
		return node.Text, nil
	}

	// If text is not attached, try to get from pages
	if node.StartIndex > 0 && node.EndIndex > 0 {
		spec := fmt.Sprintf("%d-%d", node.StartIndex, node.EndIndex)
		return GetPageContent(doc, spec)
	}

	return "", fmt.Errorf("no content available for node %q", nodeID)
}

// ParsePageSpec parses a page specification string into a sorted list of page numbers.
func ParsePageSpec(spec string) ([]int, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, fmt.Errorf("empty page spec")
	}

	const maxRangeSpan = 10000

	seen := make(map[int]bool)
	var pages []int

	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid page number %q: %w", rangeParts[0], err)
			}
			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid page number %q: %w", rangeParts[1], err)
			}
			if start < 1 {
				return nil, fmt.Errorf("page number must be positive, got %d", start)
			}
			if start > end {
				return nil, fmt.Errorf("invalid range %d-%d: start > end", start, end)
			}
			if end-start+1 > maxRangeSpan {
				return nil, fmt.Errorf("range %d-%d too large (max %d pages)", start, end, maxRangeSpan)
			}
			for p := start; p <= end; p++ {
				if !seen[p] {
					seen[p] = true
					pages = append(pages, p)
				}
			}
		} else {
			p, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid page number %q: %w", part, err)
			}
			if p < 1 {
				return nil, fmt.Errorf("page number must be positive, got %d", p)
			}
			if !seen[p] {
				seen[p] = true
				pages = append(pages, p)
			}
		}
	}

	// Sort
	for i := 0; i < len(pages); i++ {
		for j := i + 1; j < len(pages); j++ {
			if pages[j] < pages[i] {
				pages[i], pages[j] = pages[j], pages[i]
			}
		}
	}

	return pages, nil
}

func getPageByNumber(doc *tree.Document, pageNum int) (string, error) {
	for _, p := range doc.Pages {
		if p.PageNum == pageNum {
			return p.Content, nil
		}
	}
	return "", fmt.Errorf("page %d not found", pageNum)
}
