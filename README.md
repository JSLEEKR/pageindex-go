# pageindex-go

[![Go Version](https://img.shields.io/badge/Go-1.22-00ADD8?style=for-the-badge&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-131-success?style=for-the-badge)](.)
[![Original](https://img.shields.io/badge/Original-PageIndex_23.3K⭐-blue?style=for-the-badge&logo=github)](https://github.com/VectifyAI/PageIndex)

**Vectorless document indexing via hierarchical tree structures.** A Go reimplementation of [VectifyAI/PageIndex](https://github.com/VectifyAI/PageIndex) — the reasoning-based RAG approach that achieves 98.7% accuracy on FinanceBench without embeddings or vector databases.

## Why This Exists

Traditional RAG (Retrieval-Augmented Generation) relies on vector similarity search, which suffers from:

- **Semantic gap**: Query terms don't match document vocabulary
- **Lost structure**: Chunking destroys natural document sections
- **No reasoning**: Vector similarity can't follow cross-references

PageIndex takes a fundamentally different approach: build a **hierarchical tree index** of document structure, then let an LLM **reason about where to look** instead of blindly matching vectors.

This Go reimplementation adds:

- **Static binary** — zero runtime dependencies, single binary deployment
- **Goroutine concurrency** — parallel verification and summary generation
- **Type safety** — all data structures as Go structs, not runtime dicts
- **Minimal deps** — only `gopkg.in/yaml.v3` external dependency
- **Context propagation** — proper cancellation through entire pipeline

## Quick Start

### Install

```bash
go install github.com/JSLEEKR/pageindex-go/cmd/pageindex@latest
```

### Build from Source

```bash
git clone https://github.com/JSLEEKR/pageindex-go.git
cd pageindex-go
go build -o pageindex ./cmd/pageindex/
```

### Index a Markdown File

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="sk-..."

# Build an index (no LLM needed for basic markdown indexing)
pageindex build --file docs/guide.md --no-summary

# View the tree structure
pageindex show --doc guide.md

# Get specific page content
pageindex pages --doc guide.md --pages "1-5"

# Search with LLM reasoning
pageindex search --doc guide.md --query "How do I configure authentication?"
```

### Index a PDF

```bash
# Index a PDF (requires LLM for TOC detection and structure generation)
pageindex build --file report.pdf

# Show structure with JSON output
pageindex show --doc report.pdf --json

# List all indexed documents
pageindex list
```

### Configuration

Create a `config.yaml` file:

```yaml
model: "gpt-4o-2024-11-20"
retrieve_model: "gpt-4o"
base_url: "https://api.openai.com/v1"
toc_check_page_num: 20
max_page_num_each_node: 10
max_token_num_each_node: 20000
if_add_node_id: true
if_add_node_summary: true
if_add_doc_description: false
if_add_node_text: false
max_retries: 10
retry_delay_sec: 1
```

```bash
pageindex build --file report.pdf --config config.yaml
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `build` | Build an index for a PDF or Markdown file |
| `show` | Show the tree structure of an indexed document |
| `search` | Search an indexed document using LLM reasoning |
| `pages` | Get raw page/line content from an indexed document |
| `list` | List all indexed documents in workspace |
| `version` | Show version information |

### Build Options

```
--file         Path to PDF or Markdown file (required)
--config       Path to config YAML file
--workspace    Workspace directory (default: .pageindex)
--no-summary   Skip LLM summary generation
--no-id        Skip node ID assignment
--add-text     Include raw text in tree nodes
--add-desc     Generate document description via LLM
```

## Architecture

```
pageindex-go/
  cmd/pageindex/     CLI entry point
  pkg/
    tree/            Node types, tree building, traversal, serialization
    config/          YAML config loading with defaults
    llm/             LLM client (OpenAI-compatible), prompts, JSON extraction
    textract/        PDF text extraction interface
    index/           PDF pipeline, Markdown pipeline, TOC detection/verification
    retrieve/        Tool functions (get_document, get_structure, get_pages)
  internal/
    workspace/       Save/load indexed documents as JSON
```

### Core Pipeline (PDF)

The PDF pipeline automatically selects one of three processing modes:

```
Mode A: TOC with page numbers
  1. Detect TOC pages (LLM scans first 20 pages)
  2. Extract raw TOC text
  3. Transform to structured JSON entries
  4. Map TOC page numbers to physical PDF pages
  5. Calculate and apply page offset

Mode B: TOC without page numbers
  1. Same TOC detection and extraction
  2. Map entries to physical pages using content matching

Mode C: No TOC found
  1. Batch pages by token limit
  2. LLM generates hierarchical structure from content
  3. Continue generation for subsequent batches
```

After mode-specific processing:
```
4. Verify TOC accuracy (concurrent LLM checks)
5. Fix incorrect entries (up to 3 retries)
6. Convert flat entries to nested tree
7. Subdivide large nodes recursively
8. Enrich: assign IDs, attach text, generate summaries
```

### Core Pipeline (Markdown)

```
1. Extract headers (skip code blocks)
2. Assign text content to each header
3. Optional: merge small nodes (tree thinning)
4. Build nested tree using stack-based algorithm
5. Enrich: assign IDs, generate summaries
```

### Retrieval Tools

Three tool functions for agentic RAG:

```go
// Get document metadata
info, _ := retrieve.GetDocument(doc)

// Get tree structure without text (saves tokens)
structure, _ := retrieve.GetDocumentStructure(doc)

// Get raw content for specific pages
content, _ := retrieve.GetPageContent(doc, "5-7,10")

// Get content for a specific node
content, _ := retrieve.GetNodeContent(doc, "0003")
```

### Page Spec Syntax

```
"5"         Single page
"5-7"       Range (inclusive)
"3,5,7"     List
"1-3,5,7-9" Mixed ranges and singles
```

## Data Structures

### Tree Node

```json
{
  "title": "Section Name",
  "node_id": "0001",
  "start_index": 5,
  "end_index": 8,
  "summary": "LLM-generated description",
  "text": "raw page text (optional)",
  "nodes": []
}
```

### Document Container

```json
{
  "doc_name": "report.pdf",
  "doc_description": "One-sentence description",
  "structure": [],
  "pages": [{"page": 1, "content": "..."}],
  "page_count": 22
}
```

## LLM Integration

The LLM client is an interface, making it easy to mock for testing:

```go
type Client interface {
    Complete(ctx context.Context, req CompletionRequest) (string, error)
}
```

Features:
- OpenAI-compatible API (works with any compatible provider)
- Temperature 0 for deterministic output
- Automatic retry with configurable backoff (default: 10 retries, 1s delay)
- Truncation detection (finish_reason="length" triggers retry)
- JSON extraction handles: ` + "```json" + ` fences, Python None/True/False, trailing commas

## Comparison with Original

| Feature | PageIndex (Python) | pageindex-go |
|---------|-------------------|--------------|
| Language | Python 3.10+ | Go 1.22 |
| Binary size | ~50MB+ (with venv) | ~10MB static |
| Dependencies | litellm, pymupdf, PyPDF2, etc. | 1 external (yaml.v3) |
| Concurrency | asyncio | goroutines + sync |
| Type safety | Runtime dicts | Compile-time structs |
| PDF parsing | PyPDF2 + PyMuPDF | Interface-based (pluggable) |
| LLM support | 100+ via litellm | OpenAI-compatible API |
| Token counting | litellm (exact) | Estimation (~4 chars/token) |
| Config format | YAML | YAML |
| Error handling | Exceptions | Typed errors with wrapping |
| Context/cancel | Limited | Full context.Context support |

### Improvements

1. **Single binary**: No Python, no virtualenv, no pip install
2. **Type safety**: All data structures are Go structs with JSON tags
3. **Concurrent verification**: Goroutines instead of asyncio.gather
4. **Pluggable PDF**: Interface-based reader, swap implementations freely
5. **Context propagation**: Cancel any operation cleanly
6. **Error wrapping**: `fmt.Errorf("...: %w", err)` throughout

### Limitations vs Original

1. **LLM providers**: Only OpenAI-compatible APIs (vs litellm's 100+ providers)
2. **Token counting**: Estimation vs exact tokenizer
3. **PDF extraction**: Basic built-in extractor (use external library for production)
4. **Async client**: Not implemented (synchronous with goroutines instead)

## Testing

```bash
# Run all tests
go test ./... -count=1

# Run with verbose output
go test ./... -v

# Run specific package
go test ./pkg/tree/... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Test Coverage

| Package | Tests | Coverage |
|---------|-------|----------|
| tree | 22 | types, traversal, ListToTree, SetEndIndices |
| config | 8 | defaults, YAML loading, validation, env vars |
| llm | 25 | HTTP client, retry, JSON extraction, tokens |
| index | 26 | TOC detection, offset calc, markdown parsing, verification, rune truncation |
| retrieve | 25 | page spec parsing, tool functions, node content, negative page validation |
| textract | 8 | SimpleReader, PageTexts, countPages, extractTextFromPDF |
| workspace | 7 | save/load, list, delete, sanitize |
| cmd | 14 | all CLI commands, error cases |

**Total: 131 tests**

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

- Original project: [VectifyAI/PageIndex](https://github.com/VectifyAI/PageIndex) (23.3K stars)
- Paper: "PageIndex: Vectorless Reasoning-Based Document RAG"
- This is a clean-room reimplementation in Go, not a port
