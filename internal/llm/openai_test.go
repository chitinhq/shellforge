package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newOpenAITestProvider creates an OpenAIProvider pointing at the given mock server URL.
func newOpenAITestProvider(serverURL string) *OpenAIProvider {
	return NewOpenAIProvider("test-api-key", serverURL, "test-model")
}

// mockOpenAIServer creates an httptest.Server that returns the provided JSON body for
// all POST requests to /chat/completions.
func mockOpenAIServer(t *testing.T, respBody interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		b, err := json.Marshal(respBody)
		if err != nil {
			t.Errorf("marshal response: %v", err)
			return
		}
		w.Write(b)
	}))
}

// captureOpenAIServer creates a mock server that captures the decoded request body
// and returns the provided response.
func captureOpenAIServer(t *testing.T, captured *openAIRequest, respBody interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if err := json.Unmarshal(body, captured); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		b, err := json.Marshal(respBody)
		if err != nil {
			t.Errorf("marshal response: %v", err)
			return
		}
		w.Write(b)
	}))
}

// ---------------------------------------------------------------------------
// Test 1: Name()
// ---------------------------------------------------------------------------

func TestOpenAIProviderName(t *testing.T) {
	p := NewOpenAIProvider("key", "https://api.example.com/v1", "gpt-4")
	if got := p.Name(); got != "openai" {
		t.Errorf("Name() = %q, want %q", got, "openai")
	}
}

// ---------------------------------------------------------------------------
// Test 2: compile-time interface check
// ---------------------------------------------------------------------------

func TestOpenAIProviderImplementsProvider(t *testing.T) {
	var _ Provider = (*OpenAIProvider)(nil)
}

// ---------------------------------------------------------------------------
// Test 3: basic text response
// ---------------------------------------------------------------------------

func TestOpenAIChatBasic(t *testing.T) {
	apiResp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Hello, world!",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     15,
			"completion_tokens": 7,
		},
	}

	srv := mockOpenAIServer(t, apiResp)
	defer srv.Close()

	p := newOpenAITestProvider(srv.URL)
	resp, err := p.Chat([]Message{{Role: "user", Content: "Hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello, world!")
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "end_turn")
	}
	if resp.PromptTok != 15 {
		t.Errorf("PromptTok = %d, want 15", resp.PromptTok)
	}
	if resp.OutputTok != 7 {
		t.Errorf("OutputTok = %d, want 7", resp.OutputTok)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls should be empty, got %d", len(resp.ToolCalls))
	}
}

// ---------------------------------------------------------------------------
// Test 4: tool_calls response
// ---------------------------------------------------------------------------

func TestOpenAIChatWithToolCalls(t *testing.T) {
	apiResp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call_abc123",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "read_file",
								"arguments": `{"path":"/tmp/test.txt"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     20,
			"completion_tokens": 10,
		},
	}

	srv := mockOpenAIServer(t, apiResp)
	defer srv.Close()

	p := newOpenAITestProvider(srv.URL)
	resp, err := p.Chat([]Message{{Role: "user", Content: "Read that file"}}, []ToolDef{
		{
			Name:        "read_file",
			Description: "Read a file from disk",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}
	if resp.StopReason != "tool_use" {
		t.Errorf("StopReason = %q, want %q", resp.StopReason, "tool_use")
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}
	tc := resp.ToolCalls[0]
	if tc.ID != "call_abc123" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "call_abc123")
	}
	if tc.Name != "read_file" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "read_file")
	}
	if tc.Params["path"] != "/tmp/test.txt" {
		t.Errorf("ToolCall.Params[path] = %q, want %q", tc.Params["path"], "/tmp/test.txt")
	}
}

// ---------------------------------------------------------------------------
// Test 5: malformed tool call arguments — graceful degradation
// ---------------------------------------------------------------------------

func TestOpenAIChatMalformedToolCall(t *testing.T) {
	apiResp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id":   "call_bad",
							"type": "function",
							"function": map[string]interface{}{
								"name":      "broken_tool",
								"arguments": `this is not json at all`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     5,
			"completion_tokens": 3,
		},
	}

	srv := mockOpenAIServer(t, apiResp)
	defer srv.Close()

	p := newOpenAITestProvider(srv.URL)
	resp, err := p.Chat([]Message{{Role: "user", Content: "Do something"}}, nil)
	if err != nil {
		t.Fatalf("Chat() should not error on malformed tool call, got: %v", err)
	}
	if resp.StopReason != "end_turn" {
		t.Errorf("StopReason = %q, want %q (degraded)", resp.StopReason, "end_turn")
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls should be empty on degradation, got %d", len(resp.ToolCalls))
	}
	if resp.Content == "" {
		t.Error("Content should not be empty after degradation")
	}
}

// ---------------------------------------------------------------------------
// Test 6: system message sent as role "system"
// ---------------------------------------------------------------------------

func TestOpenAIChatSystemMessage(t *testing.T) {
	apiResp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "ok",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     8,
			"completion_tokens": 1,
		},
	}

	var captured openAIRequest
	srv := captureOpenAIServer(t, &captured, apiResp)
	defer srv.Close()

	p := newOpenAITestProvider(srv.URL)
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello"},
	}
	_, err := p.Chat(messages, nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if len(captured.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want 2", len(captured.Messages))
	}
	sys := captured.Messages[0]
	if sys.Role != "system" {
		t.Errorf("first message Role = %q, want %q", sys.Role, "system")
	}
	if sys.Content != "You are a helpful assistant." {
		t.Errorf("system Content = %q, want %q", sys.Content, "You are a helpful assistant.")
	}
}

// ---------------------------------------------------------------------------
// Test 7: tool_result maps to role "tool" with tool_call_id
// ---------------------------------------------------------------------------

func TestOpenAIChatToolResult(t *testing.T) {
	apiResp := map[string]interface{}{
		"choices": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Done.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]interface{}{
			"prompt_tokens":     12,
			"completion_tokens": 2,
		},
	}

	var captured openAIRequest
	srv := captureOpenAIServer(t, &captured, apiResp)
	defer srv.Close()

	p := newOpenAITestProvider(srv.URL)
	messages := []Message{
		{Role: "user", Content: "Read the file"},
		{Role: "assistant", Content: "Sure"},
		{
			Role:       "tool_result",
			Content:    "file contents here",
			ToolCallID: "call_42",
		},
	}
	_, err := p.Chat(messages, nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if len(captured.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(captured.Messages))
	}

	last := captured.Messages[2]
	if last.Role != "tool" {
		t.Errorf("tool_result Role = %q, want %q", last.Role, "tool")
	}
	if last.ToolCallID != "call_42" {
		t.Errorf("tool_result ToolCallID = %q, want %q", last.ToolCallID, "call_42")
	}
	if last.Content != "file contents here" {
		t.Errorf("tool_result Content = %q, want %q", last.Content, "file contents here")
	}
}
