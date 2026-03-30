package tree

import (
	"strings"
	"testing"
)

func buildTestTree() []*Node {
	return []*Node{
		{
			Title: "Chapter 1", NodeID: "0000", StartIndex: 1, EndIndex: 10,
			Children: []*Node{
				{Title: "Section 1.1", NodeID: "0001", StartIndex: 1, EndIndex: 5},
				{Title: "Section 1.2", NodeID: "0002", StartIndex: 6, EndIndex: 10,
					Children: []*Node{
						{Title: "Sub 1.2.1", NodeID: "0003", StartIndex: 6, EndIndex: 8},
					},
				},
			},
		},
		{Title: "Chapter 2", NodeID: "0004", StartIndex: 11, EndIndex: 20},
	}
}

func TestFlatten(t *testing.T) {
	nodes := buildTestTree()
	flat := Flatten(nodes)

	if len(flat) != 5 {
		t.Fatalf("Flatten: got %d nodes, want 5", len(flat))
	}

	expected := []string{"Chapter 1", "Section 1.1", "Section 1.2", "Sub 1.2.1", "Chapter 2"}
	for i, n := range flat {
		if n.Title != expected[i] {
			t.Errorf("flat[%d].Title = %q, want %q", i, n.Title, expected[i])
		}
	}
}

func TestLeafNodes(t *testing.T) {
	nodes := buildTestTree()
	leaves := LeafNodes(nodes)

	if len(leaves) != 3 {
		t.Fatalf("LeafNodes: got %d, want 3", len(leaves))
	}

	expected := []string{"Section 1.1", "Sub 1.2.1", "Chapter 2"}
	for i, n := range leaves {
		if n.Title != expected[i] {
			t.Errorf("leaf[%d] = %q, want %q", i, n.Title, expected[i])
		}
	}
}

func TestFindByID(t *testing.T) {
	nodes := buildTestTree()

	found := FindByID(nodes, "0003")
	if found == nil || found.Title != "Sub 1.2.1" {
		t.Error("FindByID('0003') failed")
	}

	notFound := FindByID(nodes, "9999")
	if notFound != nil {
		t.Error("FindByID('9999') should return nil")
	}
}

func TestFindByTitle(t *testing.T) {
	nodes := buildTestTree()

	found := FindByTitle(nodes, "section 1.1") // case-insensitive
	if found == nil || found.NodeID != "0001" {
		t.Error("FindByTitle case-insensitive failed")
	}

	notFound := FindByTitle(nodes, "nonexistent")
	if notFound != nil {
		t.Error("FindByTitle should return nil for missing")
	}
}

func TestAssignNodeIDs(t *testing.T) {
	nodes := []*Node{
		{Title: "A", Children: []*Node{
			{Title: "A1"},
			{Title: "A2"},
		}},
		{Title: "B"},
	}

	AssignNodeIDs(nodes)

	expected := map[string]string{
		"A": "0000", "A1": "0001", "A2": "0002", "B": "0003",
	}

	for _, n := range Flatten(nodes) {
		if want, ok := expected[n.Title]; ok {
			if n.NodeID != want {
				t.Errorf("%s: NodeID = %q, want %q", n.Title, n.NodeID, want)
			}
		}
	}
}

func TestStripText(t *testing.T) {
	nodes := []*Node{
		{Title: "A", Text: "some text", Children: []*Node{
			{Title: "B", Text: "child text"},
		}},
	}

	stripped := StripText(nodes)
	if stripped[0].Text != "" || stripped[0].Children[0].Text != "" {
		t.Error("StripText did not remove text")
	}
	// Original should be unchanged
	if nodes[0].Text != "some text" {
		t.Error("StripText modified original")
	}
}

func TestStripSummary(t *testing.T) {
	nodes := []*Node{
		{Title: "A", Summary: "summary", PrefixSummary: "prefix"},
	}
	stripped := StripSummary(nodes)
	if stripped[0].Summary != "" || stripped[0].PrefixSummary != "" {
		t.Error("StripSummary did not remove summaries")
	}
}

func TestDepth(t *testing.T) {
	tests := []struct {
		name  string
		nodes []*Node
		want  int
	}{
		{"empty", nil, 0},
		{"single", []*Node{{Title: "A"}}, 1},
		{"nested", buildTestTree(), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Depth(tt.nodes); got != tt.want {
				t.Errorf("Depth() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestListToTree(t *testing.T) {
	entries := []TOCEntry{
		{Structure: "1", Title: "Introduction", PhysicalIndex: 1},
		{Structure: "1.1", Title: "Background", PhysicalIndex: 1},
		{Structure: "1.2", Title: "Goals", PhysicalIndex: 3},
		{Structure: "2", Title: "Methods", PhysicalIndex: 5},
		{Structure: "2.1", Title: "Experiment", PhysicalIndex: 5},
		{Structure: "3", Title: "Results", PhysicalIndex: 8},
	}

	tree := ListToTree(entries)

	if len(tree) != 3 {
		t.Fatalf("ListToTree: got %d roots, want 3", len(tree))
	}

	if tree[0].Title != "Introduction" {
		t.Errorf("root[0].Title = %q, want 'Introduction'", tree[0].Title)
	}

	if len(tree[0].Children) != 2 {
		t.Fatalf("Introduction children = %d, want 2", len(tree[0].Children))
	}

	if tree[0].Children[0].Title != "Background" {
		t.Errorf("child[0] = %q, want 'Background'", tree[0].Children[0].Title)
	}

	if len(tree[1].Children) != 1 {
		t.Fatalf("Methods children = %d, want 1", len(tree[1].Children))
	}
}

func TestListToTreeEmpty(t *testing.T) {
	result := ListToTree(nil)
	if result != nil {
		t.Error("ListToTree(nil) should return nil")
	}
}

func TestListToTreeUnsorted(t *testing.T) {
	// Entries given out of order should still produce correct tree
	entries := []TOCEntry{
		{Structure: "2", Title: "B", PhysicalIndex: 5},
		{Structure: "1", Title: "A", PhysicalIndex: 1},
		{Structure: "1.1", Title: "A1", PhysicalIndex: 2},
	}

	tree := ListToTree(entries)
	if len(tree) != 2 {
		t.Fatalf("got %d roots, want 2", len(tree))
	}
	if tree[0].Title != "A" {
		t.Errorf("first root = %q, want 'A'", tree[0].Title)
	}
}

func TestStructureLevel(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{"1", 1},
		{"1.2", 2},
		{"1.2.3", 3},
		{"", 0},
	}
	for _, tt := range tests {
		if got := structureLevel(tt.code); got != tt.want {
			t.Errorf("structureLevel(%q) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func TestCompareStructureCodes(t *testing.T) {
	tests := []struct {
		a, b string
		want int // -1, 0, 1
	}{
		{"1", "2", -1},
		{"2", "1", 1},
		{"1", "1", 0},
		{"1.1", "1.2", -1},
		{"1.2", "1.1", 1},
		{"1", "1.1", -1},
		{"2", "1.9", 1},
	}
	for _, tt := range tests {
		got := compareStructureCodes(tt.a, tt.b)
		if (tt.want < 0 && got >= 0) || (tt.want > 0 && got <= 0) || (tt.want == 0 && got != 0) {
			t.Errorf("compare(%q, %q) = %d, want sign %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestSetEndIndices(t *testing.T) {
	nodes := []*Node{
		{Title: "A", StartIndex: 1, Children: []*Node{
			{Title: "A1", StartIndex: 1},
			{Title: "A2", StartIndex: 4},
		}},
		{Title: "B", StartIndex: 7},
	}

	SetEndIndices(nodes, 10)

	if nodes[0].Children[0].EndIndex != 3 {
		t.Errorf("A1 EndIndex = %d, want 3", nodes[0].Children[0].EndIndex)
	}
	if nodes[0].Children[1].EndIndex != 6 {
		t.Errorf("A2 EndIndex = %d, want 6", nodes[0].Children[1].EndIndex)
	}
	if nodes[1].EndIndex != 10 {
		t.Errorf("B EndIndex = %d, want 10", nodes[1].EndIndex)
	}
	// Parent should have max of children
	if nodes[0].EndIndex != 6 {
		t.Errorf("A EndIndex = %d, want 6", nodes[0].EndIndex)
	}
}

func TestPrintTree(t *testing.T) {
	nodes := []*Node{
		{Title: "Root", NodeID: "0000", StartIndex: 1, EndIndex: 10, Children: []*Node{
			{Title: "Child", NodeID: "0001", StartIndex: 1, EndIndex: 5},
		}},
	}
	output := PrintTree(nodes, 0)
	if output == "" {
		t.Error("PrintTree returned empty string")
	}
	if !strings.Contains(output, "Root") || !strings.Contains(output, "Child") {
		t.Error("PrintTree missing node titles")
	}
}

func TestFlattenEmpty(t *testing.T) {
	result := Flatten(nil)
	if len(result) != 0 {
		t.Errorf("Flatten(nil) returned %d items", len(result))
	}
}
