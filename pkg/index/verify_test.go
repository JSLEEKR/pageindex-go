package index

import (
	"context"
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

func TestVerifierAllCorrect(t *testing.T) {
	mock := &mockClient{responses: []string{"yes", "yes", "yes"}}
	verifier := NewVerifier(mock, "test", 5)

	entries := []tree.TOCEntry{
		{Title: "A", PhysicalIndex: 1},
		{Title: "B", PhysicalIndex: 2},
		{Title: "C", PhysicalIndex: 3},
	}
	pageTexts := []string{"A content", "B content", "C content"}

	result, err := verifier.Verify(context.Background(), entries, pageTexts)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if result.Accuracy != 1.0 {
		t.Errorf("accuracy = %f, want 1.0", result.Accuracy)
	}
	if len(result.IncorrectEntries) != 0 {
		t.Errorf("incorrect = %v", result.IncorrectEntries)
	}
}

func TestVerifierSomeIncorrect(t *testing.T) {
	mock := &mockClient{responses: []string{"yes", "no", "yes"}}
	verifier := NewVerifier(mock, "test", 1) // 1 worker to ensure order

	entries := []tree.TOCEntry{
		{Title: "A", PhysicalIndex: 1},
		{Title: "B", PhysicalIndex: 2},
		{Title: "C", PhysicalIndex: 3},
	}
	pageTexts := []string{"A", "B", "C"}

	result, err := verifier.Verify(context.Background(), entries, pageTexts)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	expectedAccuracy := 2.0 / 3.0
	if result.Accuracy < expectedAccuracy-0.01 || result.Accuracy > expectedAccuracy+0.01 {
		t.Errorf("accuracy = %f, want ~%f", result.Accuracy, expectedAccuracy)
	}
}

func TestVerifierEmptyEntries(t *testing.T) {
	mock := &mockClient{responses: []string{}}
	verifier := NewVerifier(mock, "test", 5)

	result, err := verifier.Verify(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Accuracy != 1.0 {
		t.Errorf("empty accuracy = %f", result.Accuracy)
	}
}

func TestVerifierOutOfRangeIndex(t *testing.T) {
	mock := &mockClient{responses: []string{}}
	verifier := NewVerifier(mock, "test", 5)

	entries := []tree.TOCEntry{
		{Title: "A", PhysicalIndex: 99}, // out of range
	}
	pageTexts := []string{"only page"}

	result, err := verifier.Verify(context.Background(), entries, pageTexts)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if result.Accuracy != 0 {
		t.Errorf("accuracy = %f, want 0 for out of range", result.Accuracy)
	}
}

func TestFixerFixEntries(t *testing.T) {
	mock := &mockClient{responses: []string{"3"}} // fix to page 3
	fixer := NewFixer(mock, "test", 3)

	entries := []tree.TOCEntry{
		{Title: "A", PhysicalIndex: 1},
		{Title: "B", PhysicalIndex: 2}, // incorrect
	}
	pageTexts := []string{"p1", "p2", "p3", "p4", "p5"}

	fixed, err := fixer.FixEntries(context.Background(), entries, []int{1}, pageTexts)
	if err != nil {
		t.Fatalf("FixEntries: %v", err)
	}
	if fixed[1].PhysicalIndex != 3 {
		t.Errorf("fixed[1].PhysicalIndex = %d, want 3", fixed[1].PhysicalIndex)
	}
	// Unchanged entries
	if fixed[0].PhysicalIndex != 1 {
		t.Errorf("fixed[0] changed unexpectedly")
	}
}

func TestFixerOutOfRangeIndex(t *testing.T) {
	mock := &mockClient{responses: []string{}}
	fixer := NewFixer(mock, "test", 3)

	entries := []tree.TOCEntry{{Title: "A", PhysicalIndex: 1}}

	// Index out of range should be skipped
	fixed, err := fixer.FixEntries(context.Background(), entries, []int{99}, []string{"p1"})
	if err != nil {
		t.Fatalf("FixEntries: %v", err)
	}
	if fixed[0].PhysicalIndex != 1 {
		t.Error("should not modify valid entries")
	}
}

func TestNewVerifierDefaultWorkers(t *testing.T) {
	v := NewVerifier(nil, "test", 0)
	if v.maxWorkers != 5 {
		t.Errorf("maxWorkers = %d, want 5 (default)", v.maxWorkers)
	}

	v2 := NewVerifier(nil, "test", -1)
	if v2.maxWorkers != 5 {
		t.Errorf("maxWorkers = %d for -1", v2.maxWorkers)
	}
}
