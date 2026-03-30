package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

func TestRunVersion(t *testing.T) {
	var buf bytes.Buffer
	err := run([]string{"version"}, &buf, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run version: %v", err)
	}
	if !strings.Contains(buf.String(), version) {
		t.Errorf("output = %q, should contain version", buf.String())
	}
}

func TestRunHelp(t *testing.T) {
	var buf bytes.Buffer
	err := run([]string{"help"}, &buf, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "pageindex-go") {
		t.Error("help should contain program name")
	}
}

func TestRunNoArgs(t *testing.T) {
	var buf bytes.Buffer
	err := run(nil, &buf, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Usage") {
		t.Error("no args should show usage")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"unknown"}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestCmdBuildNoFile(t *testing.T) {
	err := cmdBuild(nil, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error when --file not provided")
	}
}

func TestCmdBuildUnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.txt")
	os.WriteFile(f, []byte("hello"), 0644)

	err := cmdBuild([]string{"--file", f}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error for unsupported extension")
	}
}

func TestCmdBuildMarkdown(t *testing.T) {
	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	os.WriteFile(mdFile, []byte("# Title\n\nContent here.\n\n## Section\n\nMore content.\n"), 0644)

	workDir := filepath.Join(dir, "workspace")
	var stdout, stderr bytes.Buffer

	// This will fail on LLM calls (no API key) but should parse the markdown.
	// With no-summary and no-desc, it won't need LLM.
	err := cmdBuild([]string{
		"--file", mdFile,
		"--workspace", workDir,
		"--no-summary",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("cmdBuild: %v", err)
	}

	if !strings.Contains(stdout.String(), "Indexed test.md") {
		t.Errorf("output = %q", stdout.String())
	}

	// Verify the file was saved
	if _, err := os.Stat(filepath.Join(workDir, "test.md.json")); os.IsNotExist(err) {
		t.Error("workspace file not created")
	}
}

func TestCmdShow(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "workspace")
	os.MkdirAll(workDir, 0755)

	doc := &tree.Document{
		DocName:   "test.pdf",
		PageCount: 5,
		Structure: []*tree.Node{
			{Title: "Chapter 1", NodeID: "0000", StartIndex: 1, EndIndex: 5},
		},
	}
	data, _ := json.MarshalIndent(doc, "", "  ")
	os.WriteFile(filepath.Join(workDir, "test.pdf.json"), data, 0644)

	var buf bytes.Buffer
	err := cmdShow([]string{"--doc", "test.pdf", "--workspace", workDir}, &buf)
	if err != nil {
		t.Fatalf("cmdShow: %v", err)
	}
	if !strings.Contains(buf.String(), "Chapter 1") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdShowJSON(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "workspace")
	os.MkdirAll(workDir, 0755)

	doc := &tree.Document{
		DocName:   "test.pdf",
		Structure: []*tree.Node{{Title: "A", NodeID: "0000"}},
	}
	data, _ := json.MarshalIndent(doc, "", "  ")
	os.WriteFile(filepath.Join(workDir, "test.pdf.json"), data, 0644)

	var buf bytes.Buffer
	err := cmdShow([]string{"--doc", "test.pdf", "--workspace", workDir, "--json"}, &buf)
	if err != nil {
		t.Fatalf("cmdShow --json: %v", err)
	}
	if !strings.Contains(buf.String(), `"title"`) {
		t.Errorf("JSON output = %q", buf.String())
	}
}

func TestCmdShowNoDoc(t *testing.T) {
	err := cmdShow(nil, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error when --doc not provided")
	}
}

func TestCmdList(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "workspace")
	os.MkdirAll(workDir, 0755)

	// Create some documents
	for _, name := range []string{"doc1.pdf", "doc2.pdf"} {
		doc := &tree.Document{DocName: name, Structure: []*tree.Node{}}
		data, _ := json.MarshalIndent(doc, "", "  ")
		os.WriteFile(filepath.Join(workDir, name+".json"), data, 0644)
	}

	var buf bytes.Buffer
	err := cmdList([]string{"--workspace", workDir}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "doc1.pdf") || !strings.Contains(buf.String(), "doc2.pdf") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdListEmpty(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	err := cmdList([]string{"--workspace", dir}, &buf)
	if err != nil {
		t.Fatalf("cmdList: %v", err)
	}
	if !strings.Contains(buf.String(), "No indexed documents") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdPages(t *testing.T) {
	dir := t.TempDir()
	workDir := filepath.Join(dir, "workspace")
	os.MkdirAll(workDir, 0755)

	doc := &tree.Document{
		DocName: "test.pdf",
		Pages: []tree.Page{
			{PageNum: 1, Content: "Hello page 1"},
			{PageNum: 2, Content: "Hello page 2"},
			{PageNum: 3, Content: "Hello page 3"},
		},
		Structure: []*tree.Node{},
	}
	data, _ := json.MarshalIndent(doc, "", "  ")
	os.WriteFile(filepath.Join(workDir, "test.pdf.json"), data, 0644)

	var buf bytes.Buffer
	err := cmdPages([]string{"--doc", "test.pdf", "--pages", "1-2", "--workspace", workDir}, &buf)
	if err != nil {
		t.Fatalf("cmdPages: %v", err)
	}
	if !strings.Contains(buf.String(), "Hello page 1") {
		t.Errorf("output = %q", buf.String())
	}
}

func TestCmdPagesNoArgs(t *testing.T) {
	err := cmdPages(nil, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error when args missing")
	}
}

func TestCmdSearchNoArgs(t *testing.T) {
	err := cmdSearch(nil, &bytes.Buffer{})
	if err == nil {
		t.Error("expected error when args missing")
	}
}

func TestCountNodes(t *testing.T) {
	nodes := []*tree.Node{
		{Title: "A", Children: []*tree.Node{
			{Title: "A1"},
			{Title: "A2"},
		}},
		{Title: "B"},
	}
	if got := countNodes(nodes); got != 4 {
		t.Errorf("countNodes = %d, want 4", got)
	}
}
