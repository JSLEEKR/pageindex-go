package workspace

import (
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	doc := &tree.Document{
		DocName:        "test.pdf",
		DocDescription: "A test document",
		PageCount:      3,
		Structure: []*tree.Node{
			{
				Title:      "Chapter 1",
				NodeID:     "0000",
				StartIndex: 1,
				EndIndex:   3,
				Children: []*tree.Node{
					{Title: "Section 1.1", NodeID: "0001", StartIndex: 1, EndIndex: 2},
				},
			},
		},
		Pages: []tree.Page{
			{PageNum: 1, Content: "page 1"},
			{PageNum: 2, Content: "page 2"},
			{PageNum: 3, Content: "page 3"},
		},
	}

	if err := store.Save(doc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load("test.pdf")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.DocName != "test.pdf" {
		t.Errorf("DocName = %q", loaded.DocName)
	}
	if loaded.PageCount != 3 {
		t.Errorf("PageCount = %d", loaded.PageCount)
	}
	if len(loaded.Structure) != 1 {
		t.Fatalf("Structure len = %d", len(loaded.Structure))
	}
	if loaded.Structure[0].Title != "Chapter 1" {
		t.Errorf("root title = %q", loaded.Structure[0].Title)
	}
	if len(loaded.Structure[0].Children) != 1 {
		t.Fatalf("children = %d", len(loaded.Structure[0].Children))
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	docs := []*tree.Document{
		{DocName: "doc1.pdf", Structure: []*tree.Node{}},
		{DocName: "doc2.pdf", Structure: []*tree.Node{}},
	}
	for _, d := range docs {
		if err := store.Save(d); err != nil {
			t.Fatal(err)
		}
	}

	names, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("List returned %d, want 2", len(names))
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	doc := &tree.Document{DocName: "todelete.pdf", Structure: []*tree.Node{}}
	if err := store.Save(doc); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete("todelete.pdf"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = store.Load("todelete.pdf")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestStoreLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Load("nonexistent")
	if err == nil {
		t.Error("expected error for missing document")
	}
}

func TestStoreSaveNil(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Save(nil); err == nil {
		t.Error("expected error for nil document")
	}
}

func TestStoreDir(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if store.Dir() != dir {
		t.Errorf("Dir() = %q, want %q", store.Dir(), dir)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal.pdf", "normal.pdf"},
		{"path/to/file.pdf", "path_to_file.pdf"},
		{"file:name.pdf", "file_name.pdf"},
		{"file<>name.pdf", "file__name.pdf"},
	}
	for _, tt := range tests {
		got := sanitizeFilename(tt.input)
		if got != tt.want {
			t.Errorf("sanitize(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
