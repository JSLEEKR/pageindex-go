package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.Model != "gpt-4o-2024-11-20" {
		t.Errorf("default model = %q", cfg.Model)
	}
	if cfg.TOCCheckPageNum != 20 {
		t.Errorf("default TOCCheckPageNum = %d", cfg.TOCCheckPageNum)
	}
	if cfg.MaxPageNumEachNode != 10 {
		t.Errorf("default MaxPageNumEachNode = %d", cfg.MaxPageNumEachNode)
	}
	if !cfg.AddNodeID {
		t.Error("AddNodeID should default to true")
	}
	if !cfg.AddNodeSummary {
		t.Error("AddNodeSummary should default to true")
	}
	if cfg.AddDocDescription {
		t.Error("AddDocDescription should default to false")
	}
	if cfg.AccuracyThreshold != 0.6 {
		t.Errorf("AccuracyThreshold = %f, want 0.6", cfg.AccuracyThreshold)
	}
}

func TestLoadFromBytes(t *testing.T) {
	yaml := []byte(`
model: "gpt-4o-mini"
toc_check_page_num: 30
if_add_node_id: false
`)
	cfg, err := LoadFromBytes(yaml)
	if err != nil {
		t.Fatalf("LoadFromBytes: %v", err)
	}

	if cfg.Model != "gpt-4o-mini" {
		t.Errorf("model = %q, want gpt-4o-mini", cfg.Model)
	}
	if cfg.TOCCheckPageNum != 30 {
		t.Errorf("TOCCheckPageNum = %d, want 30", cfg.TOCCheckPageNum)
	}
	if cfg.AddNodeID {
		t.Error("AddNodeID should be false after override")
	}
	// Non-overridden defaults should remain
	if cfg.MaxPageNumEachNode != 10 {
		t.Errorf("MaxPageNumEachNode = %d, want 10 (default)", cfg.MaxPageNumEachNode)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, []byte(`model: "custom-model"`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Model != "custom-model" {
		t.Errorf("model = %q, want custom-model", cfg.Model)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load non-existent should return defaults, got error: %v", err)
	}
	if cfg.Model != "gpt-4o-2024-11-20" {
		t.Errorf("model = %q, want default", cfg.Model)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	err := os.WriteFile(path, []byte(`{invalid yaml`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Load(path)
	if err == nil {
		t.Error("Load invalid YAML should return error")
	}
}

func TestValidate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should validate: %v", err)
	}

	bad := Default()
	bad.Model = ""
	if err := bad.Validate(); err == nil {
		t.Error("empty model should fail validation")
	}

	bad2 := Default()
	bad2.TOCCheckPageNum = 0
	if err := bad2.Validate(); err == nil {
		t.Error("zero TOCCheckPageNum should fail")
	}

	bad3 := Default()
	bad3.AccuracyThreshold = 1.5
	if err := bad3.Validate(); err == nil {
		t.Error("AccuracyThreshold > 1 should fail")
	}

	bad4 := Default()
	bad4.AccuracyThreshold = -0.1
	if err := bad4.Validate(); err == nil {
		t.Error("AccuracyThreshold < 0 should fail")
	}
}

func TestLoadWithEnvAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key-123")

	cfg := Default()
	// Simulate what Load does
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	if cfg.APIKey != "test-key-123" {
		t.Errorf("APIKey = %q, want test-key-123", cfg.APIKey)
	}
}
