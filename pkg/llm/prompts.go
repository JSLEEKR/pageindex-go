package llm

// Prompt templates for LLM calls, based on the original PageIndex prompts.

const (
	// PromptTOCDetector asks the LLM to detect if a page is a table of contents.
	PromptTOCDetector = `You are analyzing a PDF document. Look at the following page content and determine if this page is part of a Table of Contents (TOC).

A TOC page typically contains:
- Chapter/section titles with page numbers
- Hierarchical listing of document contents
- References to pages in the document

Page content:
%s

Is this page a Table of Contents page? Reply with ONLY "yes" or "no".`

	// PromptTOCExtractor extracts raw TOC text from identified TOC pages.
	PromptTOCExtractor = `Extract the complete Table of Contents from the following pages. Include all entries with their page numbers and hierarchy levels.

Pages:
%s

Return the complete TOC text as-is, preserving the hierarchy and page numbers. Only return the TOC content, nothing else.`

	// PromptTOCTransformer converts raw TOC text to structured JSON.
	PromptTOCTransformer = `Convert the following Table of Contents into a structured JSON array.

Each entry should have:
- "structure": hierarchical position code (e.g., "1", "1.1", "1.2", "2", "2.1")
- "title": the section title
- "page_number": the page number listed in the TOC (0 if not available)

TOC text:
%s

Directly return the final JSON array. Do not include any explanation.`

	// PromptTOCIndexExtractor maps TOC entries to physical PDF page locations.
	PromptTOCIndexExtractor = `You are given a table of contents and the actual content of some pages from a PDF document.
Each page is wrapped with <physical_index_X> tags indicating its physical page number.

Table of Contents entries:
%s

Document pages:
%s

For each TOC entry, find which physical page it corresponds to. Match by looking for the title text appearing at the start of that physical page.

Return a JSON array where each entry has:
- "structure": the structure code from TOC
- "title": the title
- "page_number": the TOC-stated page number
- "physical_index": the physical page number (from the <physical_index_X> tag)
- "appear_start": "yes" if the title appears near the start of the page, "no" otherwise

Directly return the final JSON array.`

	// PromptPageNumberAssign assigns page numbers to TOC entries without them.
	PromptPageNumberAssign = `You are given some entries from a table of contents that need page number assignments,
and actual page content from the document with <physical_index_X> tags.

TOC entries needing page numbers:
%s

Document pages:
%s

For each TOC entry, find which physical page its content starts on.

Return a JSON array with updated entries, each having:
- "structure": the structure code
- "title": the title
- "physical_index": the physical page number where this section starts

Directly return the final JSON array.`

	// PromptGenerateTOCInit generates initial tree structure when no TOC exists.
	PromptGenerateTOCInit = `You are analyzing a document that has no Table of Contents.
Read the following pages and create a hierarchical structure for the document.

Pages:
%s

Create a JSON array representing the document structure. Each entry should have:
- "structure": hierarchical position code (e.g., "1", "1.1", "1.2", "2")
- "title": a descriptive title for the section
- "physical_index": the physical page number where this section starts (from <physical_index_X> tags)

Identify major sections and subsections based on content changes, headings, and topic shifts.

Directly return the final JSON array.`

	// PromptGenerateTOCContinue continues tree generation for subsequent page batches.
	PromptGenerateTOCContinue = `You are continuing to analyze a document that has no Table of Contents.
Here is the structure generated so far:
%s

Now read these additional pages:
%s

Continue the document structure, adding new entries for any new sections or subsections found.
Use structure codes that continue from the existing structure.

Return ONLY the new entries as a JSON array (do not repeat existing entries).

Directly return the final JSON array.`

	// PromptVerifyTOC verifies if a TOC entry correctly maps to its page.
	PromptVerifyTOC = `Verify if the following section title appears on the specified page of the document.

Section title: %s
Claimed physical page: %d

Page content:
%s

Does this title (or very similar text) appear on this page, indicating the section starts here?
Reply with ONLY "yes" or "no".`

	// PromptFixTOCEntry fixes an incorrect page mapping for a TOC entry.
	PromptFixTOCEntry = `The following section was mapped to an incorrect page. Find the correct page.

Section title: %s
Incorrectly mapped to page: %d

Here are the surrounding pages:
%s

Which physical page does this section actually start on?
Reply with ONLY the physical page number (integer).`

	// PromptGenerateSummary generates a summary for a tree node.
	PromptGenerateSummary = `Summarize the following section of a document in 1-2 sentences.

Section title: %s
Content:
%s

Provide a concise summary that captures the main topic and key points.
Reply with ONLY the summary text.`

	// PromptDocDescription generates a one-sentence document description.
	PromptDocDescription = `Based on the following document structure, provide a one-sentence description of what this document is about.

Document structure:
%s

Reply with ONLY the one-sentence description.`

	// PromptTreeSearch searches the tree for relevant nodes given a query.
	PromptTreeSearch = `You are given a question and a tree structure of a document.
Each node contains a node id, node title, and a corresponding summary.
Your task is to find all nodes that are likely to contain the answer.

Question: %s

Document tree structure:
%s

Reply in JSON format:
{"thinking": "your reasoning about which nodes to look at", "node_list": ["node_id_1", "node_id_2"]}`
)
