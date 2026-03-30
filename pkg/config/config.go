// Package config handles YAML configuration loading with sensible defaults.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for PageIndex operations.
type Config struct {
	// LLM settings
	Model         string `yaml:"model" json:"model"`
	RetrieveModel string `yaml:"retrieve_model" json:"retrieve_model"`
	APIKey        string `yaml:"api_key" json:"api_key"`
	BaseURL       string `yaml:"base_url" json:"base_url"`

	// PDF processing
	TOCCheckPageNum     int `yaml:"toc_check_page_num" json:"toc_check_page_num"`
	MaxPageNumEachNode  int `yaml:"max_page_num_each_node" json:"max_page_num_each_node"`
	MaxTokenNumEachNode int `yaml:"max_token_num_each_node" json:"max_token_num_each_node"`
	TokenLimitPerBatch  int `yaml:"token_limit_per_batch" json:"token_limit_per_batch"`

	// Feature flags
	AddNodeID          bool `yaml:"if_add_node_id" json:"if_add_node_id"`
	AddNodeSummary     bool `yaml:"if_add_node_summary" json:"if_add_node_summary"`
	AddDocDescription  bool `yaml:"if_add_doc_description" json:"if_add_doc_description"`
	AddNodeText        bool `yaml:"if_add_node_text" json:"if_add_node_text"`

	// Retry settings
	MaxRetries    int `yaml:"max_retries" json:"max_retries"`
	RetryDelaySec int `yaml:"retry_delay_sec" json:"retry_delay_sec"`

	// Verification
	MaxVerifyRetries int     `yaml:"max_verify_retries" json:"max_verify_retries"`
	AccuracyThreshold float64 `yaml:"accuracy_threshold" json:"accuracy_threshold"`

	// Markdown
	MinTokenThreshold int `yaml:"min_token_threshold" json:"min_token_threshold"`
}

// Default returns the default configuration.
func Default() *Config {
	return &Config{
		Model:               "gpt-4o-2024-11-20",
		RetrieveModel:       "gpt-4o",
		BaseURL:             "https://api.openai.com/v1",
		TOCCheckPageNum:     20,
		MaxPageNumEachNode:  10,
		MaxTokenNumEachNode: 20000,
		TokenLimitPerBatch:  20000,
		AddNodeID:           true,
		AddNodeSummary:      true,
		AddDocDescription:   false,
		AddNodeText:         false,
		MaxRetries:          10,
		RetryDelaySec:       1,
		MaxVerifyRetries:    3,
		AccuracyThreshold:   0.6,
		MinTokenThreshold:   50,
	}
}

// Load reads a YAML config file and merges it with defaults.
// If path is empty, returns defaults without reading any file.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	// Override API key from env if not set in config
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}

	return cfg, nil
}

// LoadFromBytes parses YAML config from bytes, merging with defaults.
func LoadFromBytes(data []byte) (*Config, error) {
	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	return cfg, nil
}

// Validate checks the configuration for invalid values.
func (c *Config) Validate() error {
	if c.Model == "" {
		return fmt.Errorf("model must not be empty")
	}
	if c.TOCCheckPageNum <= 0 {
		return fmt.Errorf("toc_check_page_num must be positive, got %d", c.TOCCheckPageNum)
	}
	if c.MaxPageNumEachNode <= 0 {
		return fmt.Errorf("max_page_num_each_node must be positive, got %d", c.MaxPageNumEachNode)
	}
	if c.MaxTokenNumEachNode <= 0 {
		return fmt.Errorf("max_token_num_each_node must be positive, got %d", c.MaxTokenNumEachNode)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative, got %d", c.MaxRetries)
	}
	if c.AccuracyThreshold < 0 || c.AccuracyThreshold > 1 {
		return fmt.Errorf("accuracy_threshold must be between 0 and 1, got %f", c.AccuracyThreshold)
	}
	return nil
}
