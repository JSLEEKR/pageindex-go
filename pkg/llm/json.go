package llm

import (
	"encoding/json"
	"regexp"
	"strings"
)

var (
	// jsonFenceRE matches ```json ... ``` blocks
	jsonFenceRE = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(.*?)\\s*```")

	// trailingCommaRE matches trailing commas before ] or }
	trailingCommaRE = regexp.MustCompile(`,\s*([}\]])`)

	// pythonNoneRE matches Python None values
	pythonNoneRE = regexp.MustCompile(`:\s*None\b`)

	// pythonTrueRE matches Python True values
	pythonTrueRE = regexp.MustCompile(`:\s*True\b`)

	// pythonFalseRE matches Python False values
	pythonFalseRE = regexp.MustCompile(`:\s*False\b`)

	// singleQuoteRE matches single-quoted strings (simple cases)
	singleQuoteRE = regexp.MustCompile(`'([^']*)'`)
)

// ExtractJSON attempts to extract and parse JSON from an LLM response.
// It handles: ```json fences, Python None/True/False, trailing commas, single quotes.
func ExtractJSON[T any](raw string) (T, error) {
	var result T

	cleaned := extractJSONString(raw)

	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return result, err
	}

	return result, nil
}

// extractJSONString extracts the JSON string from an LLM response,
// handling various formatting issues.
func extractJSONString(raw string) string {
	s := strings.TrimSpace(raw)

	// Try to extract from ```json fences
	if matches := jsonFenceRE.FindStringSubmatch(s); len(matches) > 1 {
		s = strings.TrimSpace(matches[1])
	}

	// If the response starts with a non-JSON character, try to find JSON
	if len(s) > 0 && s[0] != '[' && s[0] != '{' {
		// Find first [ or {
		startBracket := strings.IndexAny(s, "[{")
		if startBracket >= 0 {
			s = s[startBracket:]
		}
	}

	// Replace Python-style values
	s = pythonNoneRE.ReplaceAllString(s, ": null")
	s = pythonTrueRE.ReplaceAllString(s, ": true")
	s = pythonFalseRE.ReplaceAllString(s, ": false")

	// Remove trailing commas
	s = trailingCommaRE.ReplaceAllString(s, "$1")

	// Handle single quotes by replacing with double quotes (best effort)
	// Only do this if there are no double quotes in the value
	if !strings.Contains(s, `"`) && strings.Contains(s, `'`) {
		s = singleQuoteRE.ReplaceAllString(s, `"$1"`)
	}

	return s
}

// ExtractJSONArray extracts a JSON array from an LLM response.
func ExtractJSONArray[T any](raw string) ([]T, error) {
	return ExtractJSON[[]T](raw)
}

// MustExtractJSON extracts JSON or returns zero value (for testing).
func MustExtractJSON[T any](raw string) T {
	result, _ := ExtractJSON[T](raw)
	return result
}
