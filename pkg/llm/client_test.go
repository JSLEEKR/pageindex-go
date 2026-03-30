package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestHTTPClientComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		resp := CompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message:      struct{ Content string `json:"content"` }{Content: "hello world"},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "test-key", 0, 1)
	result, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "hello world" {
		t.Errorf("result = %q, want 'hello world'", result)
	}
}

func TestHTTPClientRetryOnServerError(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n <= 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
			return
		}
		resp := CompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message:      struct{ Content string `json:"content"` }{Content: "success"},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", 3, 0)
	result, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "success" {
		t.Errorf("result = %q", result)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("attempts = %d, want 3", atomic.LoadInt32(&attempts))
	}
}

func TestHTTPClientRetryExhausted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", 1, 0)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err == nil {
		t.Error("expected error after retries exhausted")
	}
}

func TestHTTPClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	client := NewHTTPClient(server.URL, "key", 5, 1)
	_, err := client.Complete(ctx, CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestHTTPClientTruncatedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message:      struct{ Content string `json:"content"` }{Content: "partial"},
					FinishReason: "length",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", 0, 0)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err == nil {
		t.Error("expected error for truncated response")
	}
}

func TestHTTPClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "invalid request",
				"type":    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", 0, 0)
	_, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err == nil {
		t.Error("expected API error")
	}
	if !strings.Contains(err.Error(), "invalid request") {
		t.Errorf("error should contain 'invalid request': %v", err)
	}
}

func TestHTTPClient429Retry(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		resp := CompletionResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			}{
				{
					Message:      struct{ Content string `json:"content"` }{Content: "ok"},
					FinishReason: "stop",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "key", 2, 0)
	result, err := client.Complete(context.Background(), CompletionRequest{
		Model:    "test",
		Messages: []Message{UserPrompt("hi")},
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q", result)
	}
}

func TestSystemAndUserPrompt(t *testing.T) {
	sys := SystemPrompt("you are a helper")
	if sys.Role != "system" || sys.Content != "you are a helper" {
		t.Errorf("SystemPrompt = %+v", sys)
	}

	user := UserPrompt("hello")
	if user.Role != "user" || user.Content != "hello" {
		t.Errorf("UserPrompt = %+v", user)
	}
}
