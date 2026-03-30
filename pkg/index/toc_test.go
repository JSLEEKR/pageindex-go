package index

import (
	"context"
	"sync"
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// mockClient implements llm.Client for testing.
type mockClient struct {
	responses []string
	callCount int
	mu        sync.Mutex
}

func (m *mockClient) Complete(ctx context.Context, req llm.CompletionRequest) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.responses) == 0 {
		return "", nil
	}
	if m.callCount >= len(m.responses) {
		return m.responses[len(m.responses)-1], nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return resp, nil
}

func TestTOCDetectorFindTOCPages(t *testing.T) {
	mock := &mockClient{responses: []string{"yes", "yes", "no"}}
	detector := NewTOCDetector(mock, "test", 10)

	pages := []string{"TOC page 1", "TOC page 2", "Content page", "More content"}
	tocPages, err := detector.FindTOCPages(context.Background(), pages)
	if err != nil {
		t.Fatalf("FindTOCPages: %v", err)
	}

	if len(tocPages) != 2 {
		t.Fatalf("found %d TOC pages, want 2", len(tocPages))
	}
	if tocPages[0] != 1 || tocPages[1] != 2 {
		t.Errorf("TOC pages = %v, want [1,2]", tocPages)
	}
}

func TestTOCDetectorNoTOC(t *testing.T) {
	mock := &mockClient{responses: []string{"no"}}
	detector := NewTOCDetector(mock, "test", 5)

	pages := []string{"Just content"}
	tocPages, err := detector.FindTOCPages(context.Background(), pages)
	if err != nil {
		t.Fatalf("FindTOCPages: %v", err)
	}
	if len(tocPages) != 0 {
		t.Errorf("expected no TOC pages, got %v", tocPages)
	}
}

func TestTOCDetectorRespectMaxPages(t *testing.T) {
	mock := &mockClient{responses: []string{"no", "no"}}
	detector := NewTOCDetector(mock, "test", 2)

	pages := []string{"p1", "p2", "p3", "p4", "p5"}
	_, err := detector.FindTOCPages(context.Background(), pages)
	if err != nil {
		t.Fatalf("FindTOCPages: %v", err)
	}
	if mock.callCount != 2 {
		t.Errorf("called %d times, should be limited to 2", mock.callCount)
	}
}

func TestCalculatePageOffset(t *testing.T) {
	tests := []struct {
		name    string
		entries []tree.TOCEntry
		want    int
	}{
		{
			"simple offset",
			[]tree.TOCEntry{
				{PageNumber: 1, PhysicalIndex: 5},
				{PageNumber: 2, PhysicalIndex: 6},
				{PageNumber: 3, PhysicalIndex: 7},
			},
			4,
		},
		{
			"no offset",
			[]tree.TOCEntry{
				{PageNumber: 1, PhysicalIndex: 1},
				{PageNumber: 2, PhysicalIndex: 2},
			},
			0,
		},
		{
			"mixed offsets (mode wins)",
			[]tree.TOCEntry{
				{PageNumber: 1, PhysicalIndex: 4},
				{PageNumber: 2, PhysicalIndex: 5},
				{PageNumber: 3, PhysicalIndex: 10}, // outlier
			},
			3,
		},
		{
			"no data",
			[]tree.TOCEntry{},
			0,
		},
		{
			"missing page numbers",
			[]tree.TOCEntry{
				{PageNumber: 0, PhysicalIndex: 5},
			},
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculatePageOffset(tt.entries)
			if got != tt.want {
				t.Errorf("CalculatePageOffset() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestApplyPageOffset(t *testing.T) {
	entries := []tree.TOCEntry{
		{Structure: "1", Title: "Intro", PageNumber: 1, PhysicalIndex: 0},
		{Structure: "2", Title: "Methods", PageNumber: 5, PhysicalIndex: 0},
		{Structure: "3", Title: "Results", PageNumber: 10, PhysicalIndex: 15}, // already has physical
	}

	result := ApplyPageOffset(entries, 4)

	if result[0].PhysicalIndex != 5 {
		t.Errorf("entry[0].PhysicalIndex = %d, want 5", result[0].PhysicalIndex)
	}
	if result[1].PhysicalIndex != 9 {
		t.Errorf("entry[1].PhysicalIndex = %d, want 9", result[1].PhysicalIndex)
	}
	if result[2].PhysicalIndex != 15 {
		t.Errorf("entry[2].PhysicalIndex = %d, want 15 (unchanged)", result[2].PhysicalIndex)
	}
}

func TestHasPageNumbers(t *testing.T) {
	withPages := []tree.TOCEntry{
		{PageNumber: 0},
		{PageNumber: 5},
	}
	if !HasPageNumbers(withPages) {
		t.Error("expected true for entries with page numbers")
	}

	withoutPages := []tree.TOCEntry{
		{PageNumber: 0},
		{PageNumber: 0},
	}
	if HasPageNumbers(withoutPages) {
		t.Error("expected false for entries without page numbers")
	}

	if HasPageNumbers(nil) {
		t.Error("expected false for nil")
	}
}

func TestClampPhysicalIndices(t *testing.T) {
	entries := []tree.TOCEntry{
		{PhysicalIndex: 1},
		{PhysicalIndex: 50},
		{PhysicalIndex: 0},
		{PhysicalIndex: -1},
	}

	result := ClampPhysicalIndices(entries, 20)

	if result[0].PhysicalIndex != 1 {
		t.Errorf("[0] = %d, want 1", result[0].PhysicalIndex)
	}
	if result[1].PhysicalIndex != 20 {
		t.Errorf("[1] = %d, want 20 (clamped)", result[1].PhysicalIndex)
	}
	if result[2].PhysicalIndex != 0 {
		t.Errorf("[2] = %d, want 0 (unchanged)", result[2].PhysicalIndex)
	}
	if result[3].PhysicalIndex != 1 {
		t.Errorf("[3] = %d, want 1 (clamped)", result[3].PhysicalIndex)
	}
}

func TestPDFModeString(t *testing.T) {
	tests := []struct {
		mode PDFMode
		want string
	}{
		{ModeTOCWithPages, "TOC+pages"},
		{ModeTOCNoPages, "TOC-no-pages"},
		{ModeNoTOC, "no-TOC"},
		{PDFMode(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
