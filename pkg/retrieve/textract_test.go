package retrieve

import (
	"context"
	"strings"
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/textract"
)

func TestPageTextsGet(t *testing.T) {
	pt := &textract.PageTexts{Pages: []string{"page1", "page2", "page3"}}

	text, err := pt.Get(1)
	if err != nil || text != "page1" {
		t.Errorf("Get(1) = %q, %v", text, err)
	}

	text, err = pt.Get(3)
	if err != nil || text != "page3" {
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
	pt := &textract.PageTexts{Pages: []string{"a", "b"}}
	if pt.Count() != 2 {
		t.Errorf("Count() = %d, want 2", pt.Count())
	}

	empty := &textract.PageTexts{}
	if empty.Count() != 0 {
		t.Errorf("empty Count() = %d", empty.Count())
	}
}

func TestSimpleReaderExtractAll(t *testing.T) {
	reader := textract.NewSimpleReader()
	ctx := context.Background()

	r := strings.NewReader("not a pdf")
	pages, err := reader.ExtractAll(ctx, r)
	if err != nil {
		t.Fatalf("ExtractAll: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected at least one page")
	}
}

func TestSimpleReaderExtractPage(t *testing.T) {
	reader := textract.NewSimpleReader()
	ctx := context.Background()

	r := strings.NewReader("not a pdf")
	_, err := reader.ExtractPage(ctx, r, 1)
	if err != nil {
		t.Fatalf("ExtractPage: %v", err)
	}

	r = strings.NewReader("not a pdf")
	_, err = reader.ExtractPage(ctx, r, 99)
	if err == nil {
		t.Error("expected error for out-of-range page")
	}
}

func TestNewSimpleReader(t *testing.T) {
	r := textract.NewSimpleReader()
	if r == nil {
		t.Error("NewSimpleReader returned nil")
	}
}
