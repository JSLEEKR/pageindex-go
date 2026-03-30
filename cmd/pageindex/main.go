// Package main is the CLI entry point for pageindex-go.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/JSLEEKR/pageindex-go/internal/workspace"
	"github.com/JSLEEKR/pageindex-go/pkg/config"
	"github.com/JSLEEKR/pageindex-go/pkg/index"
	"github.com/JSLEEKR/pageindex-go/pkg/llm"
	"github.com/JSLEEKR/pageindex-go/pkg/retrieve"
	"github.com/JSLEEKR/pageindex-go/pkg/tree"
)

const version = "1.0.0"

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printUsage(stdout)
		return nil
	}

	switch args[0] {
	case "build":
		return cmdBuild(args[1:], stdout, stderr)
	case "show":
		return cmdShow(args[1:], stdout)
	case "search":
		return cmdSearch(args[1:], stdout)
	case "list":
		return cmdList(args[1:], stdout)
	case "pages":
		return cmdPages(args[1:], stdout)
	case "version":
		fmt.Fprintf(stdout, "pageindex-go v%s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown command %q. Run 'pageindex help' for usage", args[0])
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `pageindex-go v%s — Vectorless document indexing via hierarchical tree structures

Usage:
  pageindex <command> [options]

Commands:
  build    Build an index for a PDF or Markdown file
  show     Show the tree structure of an indexed document
  search   Search an indexed document (requires LLM)
  list     List all indexed documents in workspace
  pages    Get page content from an indexed document
  version  Show version information
  help     Show this help message

Examples:
  pageindex build --file report.pdf
  pageindex build --file notes.md --no-summary
  pageindex show --doc report.pdf
  pageindex search --doc report.pdf --query "What are the key findings?"
  pageindex pages --doc report.pdf --pages "5-7"
  pageindex list
`, version)
}

func cmdBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)

	filePath := fs.String("file", "", "Path to PDF or Markdown file (required)")
	cfgPath := fs.String("config", "", "Path to config YAML file")
	workDir := fs.String("workspace", ".pageindex", "Workspace directory for storing indexes")
	noSummary := fs.Bool("no-summary", false, "Skip summary generation")
	noID := fs.Bool("no-id", false, "Skip node ID assignment")
	addText := fs.Bool("add-text", false, "Include raw text in nodes")
	addDesc := fs.Bool("add-desc", false, "Generate document description")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *filePath == "" {
		return fmt.Errorf("--file is required")
	}

	// Load config
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Apply CLI overrides
	if *noSummary {
		cfg.AddNodeSummary = false
	}
	if *noID {
		cfg.AddNodeID = false
	}
	if *addText {
		cfg.AddNodeText = true
	}
	if *addDesc {
		cfg.AddDocDescription = true
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Create LLM client
	client := llm.NewHTTPClient(cfg.BaseURL, cfg.APIKey, cfg.MaxRetries, cfg.RetryDelaySec)
	ctx := context.Background()

	fileName := filepath.Base(*filePath)
	ext := strings.ToLower(filepath.Ext(fileName))

	var doc *tree.Document

	switch ext {
	case ".md", ".markdown":
		content, err := os.ReadFile(*filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		pipeline := index.NewMarkdownPipeline(client, cfg)
		doc, err = pipeline.Index(ctx, fileName, string(content))
		if err != nil {
			return fmt.Errorf("indexing markdown: %w", err)
		}

		fmt.Fprintf(stdout, "Indexed %s: %d nodes, %d lines\n", fileName,
			countNodes(doc.Structure), doc.LineCount)

	case ".pdf":
		// For PDF, we'd need to extract text first
		// Using the PDF reader interface
		fmt.Fprintf(stderr, "Note: PDF text extraction requires external library.\n")
		fmt.Fprintf(stderr, "Provide pre-extracted page texts via JSON for now.\n")

		content, err := os.ReadFile(*filePath)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}

		// Try to parse as JSON array of page texts (pre-extracted)
		var pageTexts []string
		if err := json.Unmarshal(content, &pageTexts); err != nil {
			// Fall back: treat as single-page text
			pageTexts = []string{string(content)}
		}

		pipeline := index.NewPDFPipeline(client, cfg)
		result, err := pipeline.Index(ctx, fileName, pageTexts)
		if err != nil {
			return fmt.Errorf("indexing PDF: %w", err)
		}
		doc = result.Document

		fmt.Fprintf(stdout, "Indexed %s (mode: %s): %d nodes, %d pages\n",
			fileName, result.Mode, countNodes(doc.Structure), doc.PageCount)

	default:
		return fmt.Errorf("unsupported file type %q (use .pdf or .md)", ext)
	}

	// Save to workspace
	store, err := workspace.NewStore(*workDir)
	if err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	if err := store.Save(doc); err != nil {
		return fmt.Errorf("saving index: %w", err)
	}

	fmt.Fprintf(stdout, "Saved to %s/%s.json\n", *workDir, fileName)
	return nil
}

func cmdShow(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	docName := fs.String("doc", "", "Document name (required)")
	workDir := fs.String("workspace", ".pageindex", "Workspace directory")
	jsonOutput := fs.Bool("json", false, "Output as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *docName == "" {
		return fmt.Errorf("--doc is required")
	}

	store, err := workspace.NewStore(*workDir)
	if err != nil {
		return err
	}

	doc, err := store.Load(*docName)
	if err != nil {
		return fmt.Errorf("loading document: %w", err)
	}

	if *jsonOutput {
		structure, err := retrieve.GetDocumentStructure(doc)
		if err != nil {
			return err
		}
		fmt.Fprintln(stdout, structure)
	} else {
		info, _ := retrieve.GetDocument(doc)
		fmt.Fprintf(stdout, "Document: %s\n", info.Name)
		if info.Description != "" {
			fmt.Fprintf(stdout, "Description: %s\n", info.Description)
		}
		fmt.Fprintf(stdout, "Pages: %d, Nodes: %d\n\n", info.PageCount, info.NodeCount)
		fmt.Fprint(stdout, tree.PrintTree(doc.Structure, 0))
	}

	return nil
}

func cmdSearch(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	docName := fs.String("doc", "", "Document name (required)")
	query := fs.String("query", "", "Search query (required)")
	workDir := fs.String("workspace", ".pageindex", "Workspace directory")
	cfgPath := fs.String("config", "", "Config file path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *docName == "" || *query == "" {
		return fmt.Errorf("--doc and --query are required")
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}

	store, err := workspace.NewStore(*workDir)
	if err != nil {
		return err
	}

	doc, err := store.Load(*docName)
	if err != nil {
		return fmt.Errorf("loading document: %w", err)
	}

	// Get structure for search
	structureJSON, err := retrieve.GetDocumentStructure(doc)
	if err != nil {
		return err
	}

	// Use LLM to find relevant nodes
	client := llm.NewHTTPClient(cfg.BaseURL, cfg.APIKey, cfg.MaxRetries, cfg.RetryDelaySec)
	prompt := fmt.Sprintf(llm.PromptTreeSearch, *query, structureJSON)

	resp, err := client.Complete(context.Background(), llm.CompletionRequest{
		Model:       cfg.RetrieveModel,
		Messages:    []llm.Message{llm.UserPrompt(prompt)},
		Temperature: 0,
	})
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	type searchResult struct {
		Thinking string   `json:"thinking"`
		NodeList []string `json:"node_list"`
	}

	result, err := llm.ExtractJSON[searchResult](resp)
	if err != nil {
		return fmt.Errorf("parsing search result: %w", err)
	}

	fmt.Fprintf(stdout, "Query: %s\n", *query)
	fmt.Fprintf(stdout, "Reasoning: %s\n\n", result.Thinking)
	fmt.Fprintf(stdout, "Relevant nodes:\n")

	for _, nodeID := range result.NodeList {
		node := tree.FindByID(doc.Structure, nodeID)
		if node != nil {
			fmt.Fprintf(stdout, "  [%s] %s", nodeID, node.Title)
			if node.StartIndex > 0 {
				fmt.Fprintf(stdout, " (pp. %d-%d)", node.StartIndex, node.EndIndex)
			}
			fmt.Fprintln(stdout)
		}
	}

	return nil
}

func cmdList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	workDir := fs.String("workspace", ".pageindex", "Workspace directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := workspace.NewStore(*workDir)
	if err != nil {
		return err
	}

	names, err := store.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Fprintln(stdout, "No indexed documents found.")
		return nil
	}

	fmt.Fprintf(stdout, "Indexed documents (%d):\n", len(names))
	for _, name := range names {
		fmt.Fprintf(stdout, "  %s\n", name)
	}

	return nil
}

func cmdPages(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("pages", flag.ContinueOnError)
	docName := fs.String("doc", "", "Document name (required)")
	pages := fs.String("pages", "", "Page spec e.g. '1-3,5' (required)")
	workDir := fs.String("workspace", ".pageindex", "Workspace directory")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *docName == "" || *pages == "" {
		return fmt.Errorf("--doc and --pages are required")
	}

	store, err := workspace.NewStore(*workDir)
	if err != nil {
		return err
	}

	doc, err := store.Load(*docName)
	if err != nil {
		return fmt.Errorf("loading document: %w", err)
	}

	content, err := retrieve.GetPageContent(doc, *pages)
	if err != nil {
		return err
	}

	fmt.Fprintln(stdout, content)
	return nil
}

func countNodes(nodes []*tree.Node) int {
	count := len(nodes)
	for _, n := range nodes {
		count += countNodes(n.Children)
	}
	return count
}
