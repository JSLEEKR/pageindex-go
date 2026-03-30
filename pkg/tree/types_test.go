package tree

import (
	"testing"
)

func TestNodePageSpan(t *testing.T) {
	tests := []struct {
		name  string
		node  Node
		want  int
	}{
		{"normal range", Node{StartIndex: 1, EndIndex: 5}, 5},
		{"single page", Node{StartIndex: 3, EndIndex: 3}, 1},
		{"no indices", Node{}, 0},
		{"only start", Node{StartIndex: 1}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.node.PageSpan(); got != tt.want {
				t.Errorf("PageSpan() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNodeIsLeaf(t *testing.T) {
	leaf := &Node{Title: "leaf"}
	parent := &Node{Title: "parent", Children: []*Node{leaf}}

	if !leaf.IsLeaf() {
		t.Error("expected leaf to be leaf")
	}
	if parent.IsLeaf() {
		t.Error("expected parent to not be leaf")
	}
}

func TestNodeClone(t *testing.T) {
	original := &Node{
		Title:      "root",
		StartIndex: 1,
		EndIndex:   10,
		Children: []*Node{
			{Title: "child1", StartIndex: 1, EndIndex: 5},
			{Title: "child2", StartIndex: 6, EndIndex: 10},
		},
	}

	clone := original.Clone()

	if clone.Title != original.Title {
		t.Errorf("clone title = %q, want %q", clone.Title, original.Title)
	}

	// Modify clone should not affect original
	clone.Title = "modified"
	clone.Children[0].Title = "modified-child"

	if original.Title == "modified" {
		t.Error("modifying clone affected original title")
	}
	if original.Children[0].Title == "modified-child" {
		t.Error("modifying clone child affected original child")
	}
}

func TestNodeCloneNil(t *testing.T) {
	var n *Node
	if n.Clone() != nil {
		t.Error("Clone of nil should return nil")
	}
}

func TestNodeTotalDescendants(t *testing.T) {
	root := &Node{
		Title: "root",
		Children: []*Node{
			{Title: "a", Children: []*Node{
				{Title: "a1"},
				{Title: "a2"},
			}},
			{Title: "b"},
		},
	}
	if got := root.TotalDescendants(); got != 4 {
		t.Errorf("TotalDescendants() = %d, want 4", got)
	}

	leaf := &Node{Title: "leaf"}
	if got := leaf.TotalDescendants(); got != 0 {
		t.Errorf("leaf TotalDescendants() = %d, want 0", got)
	}
}
