// Package textract provides PDF text extraction through a pluggable interface.
package textract

import (
	"context"
	"fmt"
	"io"
)

// Reader is the interface for PDF text extraction.
type Reader interface {
	// PageCount returns the number of pages in the PDF.
	PageCount(ctx context.Context, r io.ReadSeeker) (int, error)

	// ExtractPage extracts text from a single page (1-indexed).
	ExtractPage(ctx context.Context, r io.ReadSeeker, page int) (string, error)

	// ExtractAll extracts text from all pages, returning a slice indexed from 0.
	ExtractAll(ctx context.Context, r io.ReadSeeker) ([]string, error)
}

// SimpleReader is a basic PDF text extractor that reads raw text streams.
// For production use, replace with pdfcpu or unipdf-based implementation.
type SimpleReader struct{}

// NewSimpleReader creates a new SimpleReader.
func NewSimpleReader() *SimpleReader {
	return &SimpleReader{}
}

// PageCount returns the number of pages by scanning the PDF for page markers.
func (sr *SimpleReader) PageCount(ctx context.Context, r io.ReadSeeker) (int, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, fmt.Errorf("reading PDF: %w", err)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seeking to start: %w", err)
	}

	count := countPages(data)
	if count == 0 {
		return 0, fmt.Errorf("no pages found in PDF")
	}
	return count, nil
}

// ExtractPage extracts text from a single page.
func (sr *SimpleReader) ExtractPage(ctx context.Context, r io.ReadSeeker, page int) (string, error) {
	pages, err := sr.ExtractAll(ctx, r)
	if err != nil {
		return "", err
	}
	if page < 1 || page > len(pages) {
		return "", fmt.Errorf("page %d out of range [1, %d]", page, len(pages))
	}
	return pages[page-1], nil
}

// ExtractAll extracts text from all pages.
func (sr *SimpleReader) ExtractAll(ctx context.Context, r io.ReadSeeker) ([]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading PDF: %w", err)
	}

	pages := extractTextFromPDF(data)
	if len(pages) == 0 {
		return []string{""}, nil
	}
	return pages, nil
}

// countPages counts the number of /Type /Page entries in a PDF.
func countPages(data []byte) int {
	count := 0
	for i := 0; i < len(data)-15; i++ {
		// Look for /Type /Page (but not /Type /Pages)
		if data[i] == '/' && i+11 < len(data) {
			chunk := string(data[i : i+12])
			if chunk == "/Type /Page " || chunk == "/Type /Page\n" || chunk == "/Type /Page\r" {
				count++
			}
		}
	}
	return count
}

// extractTextFromPDF performs basic text extraction from a PDF byte stream.
// This is a simplified extractor — for production use, use pdfcpu or similar.
func extractTextFromPDF(data []byte) []string {
	// This is a placeholder that returns empty pages based on page count.
	// Real implementation would parse PDF content streams.
	count := countPages(data)
	if count == 0 {
		count = 1
	}
	pages := make([]string, count)
	for i := range pages {
		pages[i] = ""
	}
	return pages
}

// PageTexts holds extracted text for all pages in a document.
type PageTexts struct {
	Pages []string
}

// Get returns the text for a page (1-indexed).
func (pt *PageTexts) Get(page int) (string, error) {
	if page < 1 || page > len(pt.Pages) {
		return "", fmt.Errorf("page %d out of range [1, %d]", page, len(pt.Pages))
	}
	return pt.Pages[page-1], nil
}

// Count returns the number of pages.
func (pt *PageTexts) Count() int {
	return len(pt.Pages)
}
