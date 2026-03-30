package llm

import (
	"testing"
)

func TestExtractJSONFromFence(t *testing.T) {
	raw := "Here is the result:\n```json\n[{\"title\": \"hello\"}]\n```\nDone."

	type item struct {
		Title string `json:"title"`
	}

	result, err := ExtractJSON[[]item](raw)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if len(result) != 1 || result[0].Title != "hello" {
		t.Errorf("result = %+v", result)
	}
}

func TestExtractJSONPlain(t *testing.T) {
	raw := `[{"structure": "1", "title": "Intro"}]`

	type entry struct {
		Structure string `json:"structure"`
		Title     string `json:"title"`
	}

	result, err := ExtractJSON[[]entry](raw)
	if err != nil {
		t.Fatalf("ExtractJSON: %v", err)
	}
	if len(result) != 1 || result[0].Title != "Intro" {
		t.Errorf("result = %+v", result)
	}
}

func TestExtractJSONTrailingComma(t *testing.T) {
	raw := `[{"title": "A",}, {"title": "B",}]`

	type item struct {
		Title string `json:"title"`
	}

	result, err := ExtractJSON[[]item](raw)
	if err != nil {
		t.Fatalf("ExtractJSON with trailing commas: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

func TestExtractJSONPythonNone(t *testing.T) {
	raw := `{"title": "Test", "summary": None, "active": True, "deleted": False}`

	type item struct {
		Title   string  `json:"title"`
		Summary *string `json:"summary"`
		Active  bool    `json:"active"`
		Deleted bool    `json:"deleted"`
	}

	result, err := ExtractJSON[item](raw)
	if err != nil {
		t.Fatalf("ExtractJSON with Python None: %v", err)
	}
	if result.Title != "Test" {
		t.Errorf("title = %q", result.Title)
	}
	if result.Summary != nil {
		t.Error("summary should be nil")
	}
	if !result.Active {
		t.Error("active should be true")
	}
	if result.Deleted {
		t.Error("deleted should be false")
	}
}

func TestExtractJSONWithPreamble(t *testing.T) {
	raw := `Sure, here is the JSON:

[{"structure": "1", "title": "First"}]`

	type entry struct {
		Structure string `json:"structure"`
		Title     string `json:"title"`
	}

	result, err := ExtractJSON[[]entry](raw)
	if err != nil {
		t.Fatalf("ExtractJSON with preamble: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len = %d", len(result))
	}
}

func TestExtractJSONObject(t *testing.T) {
	raw := `{"thinking": "looking at nodes", "node_list": ["0001", "0003"]}`

	type searchResult struct {
		Thinking string   `json:"thinking"`
		NodeList []string `json:"node_list"`
	}

	result, err := ExtractJSON[searchResult](raw)
	if err != nil {
		t.Fatalf("ExtractJSON object: %v", err)
	}
	if result.Thinking != "looking at nodes" {
		t.Errorf("thinking = %q", result.Thinking)
	}
	if len(result.NodeList) != 2 {
		t.Errorf("node_list len = %d", len(result.NodeList))
	}
}

func TestExtractJSONInvalid(t *testing.T) {
	_, err := ExtractJSON[map[string]string]("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestExtractJSONEmpty(t *testing.T) {
	_, err := ExtractJSON[[]string]("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}

func TestExtractJSONFenceNoLanguage(t *testing.T) {
	raw := "```\n{\"key\": \"value\"}\n```"

	result, err := ExtractJSON[map[string]string](raw)
	if err != nil {
		t.Fatalf("ExtractJSON fence no lang: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("key = %q", result["key"])
	}
}

func TestExtractJSONNestedTrailingComma(t *testing.T) {
	raw := `{"items": [{"a": 1,}, {"b": 2,},],}`

	type data struct {
		Items []map[string]int `json:"items"`
	}

	result, err := ExtractJSON[data](raw)
	if err != nil {
		t.Fatalf("nested trailing comma: %v", err)
	}
	if len(result.Items) != 2 {
		t.Errorf("items len = %d", len(result.Items))
	}
}

func TestExtractJSONArray(t *testing.T) {
	raw := `["hello", "world"]`
	result, err := ExtractJSONArray[string](raw)
	if err != nil {
		t.Fatalf("ExtractJSONArray: %v", err)
	}
	if len(result) != 2 || result[0] != "hello" {
		t.Errorf("result = %v", result)
	}
}
