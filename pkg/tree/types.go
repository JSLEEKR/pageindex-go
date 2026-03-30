// Package tree defines the core data structures for PageIndex tree nodes and documents.
package tree

import "encoding/json"

// Node represents a single node in the document tree hierarchy.
type Node struct {
	Title         string `json:"title"`
	NodeID        string `json:"node_id,omitempty"`
	StartIndex    int    `json:"start_index,omitempty"`    // physical page number (1-indexed, PDF)
	EndIndex      int    `json:"end_index,omitempty"`      // inclusive end page (PDF)
	LineNum       int    `json:"line_num,omitempty"`        // line number (Markdown)
	Summary       string `json:"summary,omitempty"`
	PrefixSummary string `json:"prefix_summary,omitempty"`
	Text          string `json:"text,omitempty"`
	Children      []*Node `json:"nodes,omitempty"`
}

// TOCEntry represents a flat table-of-contents entry before tree conversion.
type TOCEntry struct {
	Structure     string `json:"structure"`      // hierarchical position code e.g. "1.2.3"
	Title         string `json:"title"`
	PhysicalIndex int    `json:"physical_index"`  // physical page in PDF
	PageNumber    int    `json:"page_number"`     // TOC-stated page number
	AppearStart   string `json:"appear_start"`    // "yes" or "no"
}

// Page holds the text content of a single PDF page.
type Page struct {
	PageNum int    `json:"page"`
	Content string `json:"content"`
}

// Document is the top-level container for an indexed document.
type Document struct {
	DocName        string  `json:"doc_name"`
	DocDescription string  `json:"doc_description,omitempty"`
	Structure      []*Node `json:"structure"`
	Pages          []Page  `json:"pages,omitempty"`
	PageCount      int     `json:"page_count,omitempty"`
	LineCount      int     `json:"line_count,omitempty"`
}

// Clone returns a deep copy of the node and all children.
func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}
	c := *n
	if n.Children != nil {
		c.Children = make([]*Node, len(n.Children))
		for i, ch := range n.Children {
			c.Children[i] = ch.Clone()
		}
	}
	return &c
}

// MarshalJSON implements custom JSON marshaling to maintain field order
// consistent with the original Python output.
func (d *Document) MarshalJSON() ([]byte, error) {
	type Alias Document
	return json.Marshal((*Alias)(d))
}

// PageSpan returns the number of pages this node spans.
func (n *Node) PageSpan() int {
	if n.EndIndex <= 0 || n.StartIndex <= 0 {
		return 0
	}
	return n.EndIndex - n.StartIndex + 1
}

// IsLeaf returns true if the node has no children.
func (n *Node) IsLeaf() bool {
	return len(n.Children) == 0
}

// TotalDescendants returns the total number of descendant nodes.
func (n *Node) TotalDescendants() int {
	count := 0
	for _, ch := range n.Children {
		count += 1 + ch.TotalDescendants()
	}
	return count
}
