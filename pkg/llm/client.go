// Package llm provides an LLM client for OpenAI-compatible APIs.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the interface for LLM completion calls.
type Client interface {
	// Complete sends a chat completion request and returns the response text.
	Complete(ctx context.Context, req CompletionRequest) (string, error)
}

// CompletionRequest holds the parameters for a chat completion.
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
}

// Message represents a single chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionResponse is the raw OpenAI API response structure.
type CompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// HTTPClient implements Client using net/http to call OpenAI-compatible APIs.
type HTTPClient struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	MaxRetries int
	RetryDelay time.Duration
}

// NewHTTPClient creates a new LLM client.
func NewHTTPClient(baseURL, apiKey string, maxRetries int, retryDelaySec int) *HTTPClient {
	return &HTTPClient{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
		MaxRetries: maxRetries,
		RetryDelay: time.Duration(retryDelaySec) * time.Second,
	}
}

// Complete sends a chat completion request with retry logic.
func (c *HTTPClient) Complete(ctx context.Context, req CompletionRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	url := c.BaseURL + "/chat/completions"

	var lastErr error
	for attempt := 0; attempt <= c.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(c.RetryDelay):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "", fmt.Errorf("creating request: %w", err)
		}

		httpReq.Header.Set("Content-Type", "application/json")
		if c.APIKey != "" {
			httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
		}

		resp, err := c.HTTPClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt+1, err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("attempt %d reading response: %w", attempt+1, err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("attempt %d: HTTP %d: %s", attempt+1, resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		var completionResp CompletionResponse
		if err := json.Unmarshal(respBody, &completionResp); err != nil {
			lastErr = fmt.Errorf("attempt %d unmarshaling: %w", attempt+1, err)
			continue
		}

		if completionResp.Error != nil {
			return "", fmt.Errorf("API error: %s", completionResp.Error.Message)
		}

		if len(completionResp.Choices) == 0 {
			lastErr = fmt.Errorf("attempt %d: no choices in response", attempt+1)
			continue
		}

		content := completionResp.Choices[0].Message.Content

		// Check for truncation
		if completionResp.Choices[0].FinishReason == "length" {
			lastErr = fmt.Errorf("attempt %d: response truncated (finish_reason=length)", attempt+1)
			continue
		}

		return content, nil
	}

	return "", fmt.Errorf("all %d attempts failed, last error: %w", c.MaxRetries+1, lastErr)
}

// SystemPrompt creates a system message.
func SystemPrompt(content string) Message {
	return Message{Role: "system", Content: content}
}

// UserPrompt creates a user message.
func UserPrompt(content string) Message {
	return Message{Role: "user", Content: content}
}
