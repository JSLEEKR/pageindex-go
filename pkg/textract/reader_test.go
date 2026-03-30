package textract

import (
	"context"
	"strings"
	"testing"
)

func TestSimpleReaderPageCount(t *testing.T) {
	r := NewSimpleReader()
	ctx := context.Background()

	// Non-PDF data — no /Type /Page markers
	reader := strings.NewReader("not a pdf")
	_, err := r.PageCount(ctx, reader)
	if err == nil {
		t.Error("expected error for non-PDF data without page markers")
	}
}

func TestSimpleReaderExtractAll(t *testing.T) {
	r := NewSimpleReader()
	ctx := context.Background()

	reader := strings.NewReader("not a pdf")
	pages, err := r.ExtractAll(ctx, reader)
	if err != nil {
		t.Fatalf("ExtractAll: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected at least one page")
	}
}

func TestSimpleReaderExtractPage(t *testing.T) {
	r := NewSimpleReader()
	ctx := context.Background()

	reader := strings.NewReader("not a pdf")
	_, err := r.ExtractPage(ctx, reader, 1)
	if err != nil {
		t.Fatalf("ExtractPage(1): %v", err)
	}

	reader = strings.NewReader("not a pdf")
	_, err = r.ExtractPage(ctx, reader, 99)
	if err == nil {
		t.Error("expected error for out-of-range page")
	}

	reader = strings.NewReader("not a pdf")
	_, err = r.ExtractPage(ctx, reader, 0)
	if err == nil {
		t.Error("expected error for page 0")
	}
}

func TestPageTextsGet(t *testing.T) {
	pt := &PageTexts{Pages: []string{"a", "b", "c"}}

	text, err := pt.Get(1)
	if err != nil || text != "a" {
		t.Errorf("Get(1) = %q, %v", text, err)
	}

	text, err = pt.Get(3)
	if err != nil || text != "c" {
		t.Errorf("Get(3) = %q, %v", text, err)
	}

	_, err = pt.Get(0)
	if err == nil {
		t.Error("Get(0) should error")
	}

	_, err = pt.Get(4)
	if err == nil {
		t.Error("Get(4) should error")
	}
}

func TestPageTextsCount(t *testing.T) {
	pt := &PageTexts{Pages: []string{"a", "b"}}
	if pt.Count() != 2 {
		t.Errorf("Count() = %d, want 2", pt.Count())
	}

	empty := &PageTexts{}
	if empty.Count() != 0 {
		t.Errorf("empty Count() = %d", empty.Count())
	}
}

func TestCountPages(t *testing.T) {
	// No markers
	if got := countPages([]byte("no markers here")); got != 0 {
		t.Errorf("countPages(no markers) = %d", got)
	}

	// With /Type /Page markers (not /Type /Pages)
	// Note: countPages requires at least 15 bytes after start position,
	// so put enough padding at end
	data := []byte("blah /Type /Page blah /Type /Pages blah /Type /Page               ")
	got := countPages(data)
	if got != 2 {
		t.Errorf("countPages = %d, want 2", got)
	}
}

func TestExtractTextFromPDF(t *testing.T) {
	// Non-PDF data returns at least one empty page
	pages := extractTextFromPDF([]byte("not a pdf"))
	if len(pages) != 1 {
		t.Errorf("got %d pages, want 1", len(pages))
	}
}

func TestNewSimpleReader(t *testing.T) {
	r := NewSimpleReader()
	if r == nil {
		t.Error("NewSimpleReader returned nil")
	}
}
