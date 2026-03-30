// Package workspace handles saving and loading indexed documents.
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

// Store manages document persistence in a workspace directory.
type Store struct {
	dir string
}

// NewStore creates a new workspace store.
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating workspace dir: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Save serializes a document to a JSON file in the workspace.
func (s *Store) Save(doc *tree.Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	name := sanitizeFilename(doc.DocName) + ".json"
	path := filepath.Join(s.dir, name)

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling document: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

// Load reads a document from a JSON file in the workspace.
func (s *Store) Load(name string) (*tree.Document, error) {
	fileName := sanitizeFilename(name) + ".json"
	path := filepath.Join(s.dir, fileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var doc tree.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &doc, nil
}

// List returns the names of all documents in the workspace.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("reading workspace dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			name := e.Name()
			names = append(names, name[:len(name)-5]) // strip .json
		}
	}

	return names, nil
}

// Delete removes a document from the workspace.
func (s *Store) Delete(name string) error {
	fileName := sanitizeFilename(name) + ".json"
	path := filepath.Join(s.dir, fileName)

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("deleting %s: %w", path, err)
	}
	return nil
}

// Dir returns the workspace directory path.
func (s *Store) Dir() string {
	return s.dir
}

func sanitizeFilename(name string) string {
	// Replace common problematic characters
	var sb strings.Builder
	sb.Grow(len(name))
	for _, r := range name {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			sb.WriteByte('_')
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
