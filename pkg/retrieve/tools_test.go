package retrieve

import (
	"strings"
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

func buildTestDoc() *tree.Document {
	return &tree.Document{
		DocName:        "test.pdf",
		DocDescription: "A test document",
		PageCount:      5,
		Structure: []*tree.Node{
			{
				Title:      "Chapter 1",
				NodeID:     "0000",
				StartIndex: 1,
				EndIndex:   3,
				Text:       "Chapter 1 content",
				Children: []*tree.Node{
					{
						Title:      "Section 1.1",
						NodeID:     "0001",
						StartIndex: 1,
						EndIndex:   2,
						Text:       "Section 1.1 content",
					},
				},
			},
			{
				Title:      "Chapter 2",
				NodeID:     "0002",
				StartIndex: 4,
				EndIndex:   5,
				Text:       "Chapter 2 content",
			},
		},
		Pages: []tree.Page{
			{PageNum: 1, Content: "Page 1 text"},
			{PageNum: 2, Content: "Page 2 text"},
			{PageNum: 3, Content: "Page 3 text"},
			{PageNum: 4, Content: "Page 4 text"},
			{PageNum: 5, Content: "Page 5 text"},
		},
	}
}

func TestGetDocument(t *testing.T) {
	doc := buildTestDoc()
	info, err := GetDocument(doc)
	if err != nil {
		t.Fatalf("GetDocument: %v", err)
	}

	if info.Name != "test.pdf" {
		t.Errorf("Name = %q", info.Name)
	}
	if info.Description != "A test document" {
		t.Errorf("Description = %q", info.Description)
	}
	if info.PageCount != 5 {
		t.Errorf("PageCount = %d", info.PageCount)
	}
	if info.NodeCount != 3 {
		t.Errorf("NodeCount = %d, want 3", info.NodeCount)
	}
}

func TestGetDocumentNil(t *testing.T) {
	_, err := GetDocument(nil)
	if err == nil {
		t.Error("expected error for nil document")
	}
}

func TestGetDocumentStructure(t *testing.T) {
	doc := buildTestDoc()
	structure, err := GetDocumentStructure(doc)
	if err != nil {
		t.Fatalf("GetDocumentStructure: %v", err)
	}

	if !strings.Contains(structure, "Chapter 1") {
		t.Error("structure should contain 'Chapter 1'")
	}
	if strings.Contains(structure, "Chapter 1 content") {
		t.Error("structure should not contain text content")
	}
	if !strings.Contains(structure, "0000") {
		t.Error("structure should contain node IDs")
	}
}

func TestGetDocumentStructureNil(t *testing.T) {
	_, err := GetDocumentStructure(nil)
	if err == nil {
		t.Error("expected error for nil document")
	}
}

func TestGetPageContent(t *testing.T) {
	doc := buildTestDoc()

	// Single page
	content, err := GetPageContent(doc, "1")
	if err != nil {
		t.Fatalf("GetPageContent(1): %v", err)
	}
	if !strings.Contains(content, "Page 1 text") {
		t.Errorf("content = %q", content)
	}

	// Range
	content, err = GetPageContent(doc, "2-4")
	if err != nil {
		t.Fatalf("GetPageContent(2-4): %v", err)
	}
	if !strings.Contains(content, "Page 2") || !strings.Contains(content, "Page 4") {
		t.Errorf("range content missing pages: %q", content)
	}

	// List
	content, err = GetPageContent(doc, "1,3,5")
	if err != nil {
		t.Fatalf("GetPageContent(1,3,5): %v", err)
	}
	if !strings.Contains(content, "Page 1") || !strings.Contains(content, "Page 5") {
		t.Errorf("list content missing pages: %q", content)
	}

	// Mixed
	content, err = GetPageContent(doc, "1-2,5")
	if err != nil {
		t.Fatalf("GetPageContent(1-2,5): %v", err)
	}
	if !strings.Contains(content, "Page 1") || !strings.Contains(content, "Page 5") {
		t.Error("mixed content missing pages")
	}
}

func TestGetPageContentErrors(t *testing.T) {
	doc := buildTestDoc()

	_, err := GetPageContent(doc, "99")
	if err == nil {
		t.Error("expected error for missing page")
	}

	_, err = GetPageContent(doc, "")
	if err == nil {
		t.Error("expected error for empty spec")
	}

	_, err = GetPageContent(nil, "1")
	if err == nil {
		t.Error("expected error for nil document")
	}
}

func TestGetNodeContent(t *testing.T) {
	doc := buildTestDoc()

	content, err := GetNodeContent(doc, "0001")
	if err != nil {
		t.Fatalf("GetNodeContent: %v", err)
	}
	if content != "Section 1.1 content" {
		t.Errorf("content = %q", content)
	}
}

func TestGetNodeContentNotFound(t *testing.T) {
	doc := buildTestDoc()
	_, err := GetNodeContent(doc, "9999")
	if err == nil {
		t.Error("expected error for missing node")
	}
}

func TestGetNodeContentNil(t *testing.T) {
	_, err := GetNodeContent(nil, "0000")
	if err == nil {
		t.Error("expected error for nil doc")
	}
}

func TestGetNodeContentFromPages(t *testing.T) {
	doc := &tree.Document{
		DocName:   "test.pdf",
		PageCount: 3,
		Structure: []*tree.Node{
			{Title: "A", NodeID: "0000", StartIndex: 1, EndIndex: 2},
		},
		Pages: []tree.Page{
			{PageNum: 1, Content: "p1"},
			{PageNum: 2, Content: "p2"},
			{PageNum: 3, Content: "p3"},
		},
	}

	content, err := GetNodeContent(doc, "0000")
	if err != nil {
		t.Fatalf("GetNodeContent: %v", err)
	}
	if !strings.Contains(content, "p1") || !strings.Contains(content, "p2") {
		t.Errorf("content = %q, should contain p1 and p2", content)
	}
}

func TestParsePageSpec(t *testing.T) {
	tests := []struct {
		spec string
		want []int
		err  bool
	}{
		{"1", []int{1}, false},
		{"1-3", []int{1, 2, 3}, false},
		{"1,3,5", []int{1, 3, 5}, false},
		{"1-2,5,7-9", []int{1, 2, 5, 7, 8, 9}, false},
		{"5,3,1", []int{1, 3, 5}, false}, // sorted
		{"1,1,2", []int{1, 2}, false},     // deduped
		{"", nil, true},
		{"abc", nil, true},
		{"3-1", nil, true}, // invalid range
		{"1-abc", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.spec, func(t *testing.T) {
			pages, err := ParsePageSpec(tt.spec)
			if tt.err {
				if err == nil {
					t.Errorf("expected error for %q", tt.spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePageSpec(%q): %v", tt.spec, err)
			}
			if len(pages) != len(tt.want) {
				t.Fatalf("got %v, want %v", pages, tt.want)
			}
			for i := range pages {
				if pages[i] != tt.want[i] {
					t.Errorf("page[%d] = %d, want %d", i, pages[i], tt.want[i])
				}
			}
		})
	}
}

func TestParsePageSpecWithSpaces(t *testing.T) {
	pages, err := ParsePageSpec(" 1 - 3 , 5 ")
	if err != nil {
		t.Fatalf("ParsePageSpec: %v", err)
	}
	if len(pages) != 4 {
		t.Errorf("got %v", pages)
	}
}

func TestParsePageSpecNegativeNumbers(t *testing.T) {
	_, err := ParsePageSpec("-1")
	if err == nil {
		t.Error("expected error for negative page number")
	}

	_, err = ParsePageSpec("0")
	if err == nil {
		t.Error("expected error for zero page number")
	}

	_, err = ParsePageSpec("0-5")
	if err == nil {
		t.Error("expected error for range starting at 0")
	}
}

func TestParsePageSpecHugeRange(t *testing.T) {
	_, err := ParsePageSpec("1-1000000")
	if err == nil {
		t.Error("expected error for excessively large range")
	}
}
