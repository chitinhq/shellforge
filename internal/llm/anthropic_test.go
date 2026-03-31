package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newAnthropicTestProvider creates an AnthropicProvider pointing at the given mock server URL.
func newAnthropicTestProvider(serverURL string) *AnthropicProvider {
	p := NewAnthropicProvider("test-api-key", "test-model")
	p.baseURL = serverURL
	return p
}

// mustMarshal marshals v to JSON and panics on error (test helper).
func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// mockServer creates an httptest.Server that returns the provided JSON body for
// all POST requests to /v1/messages.
func mockServer(t *testing.T, respBody any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(mustMarshal(respBody))
	}))
}

// captureServer creates a mock server that captures the decoded request body
// and returns the provided response.
func captureServer(t *testing.T, captured *anthropicRequest, respBody any) *httptest.Server {
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
		w.Write(mustMarshal(respBody))
	}))
}

// ---------------------------------------------------------------------------
// Test 1: Name()
// ---------------------------------------------------------------------------

func TestAnthropicProviderName(t *testing.T) {
	p := NewAnthropicProvider("key", "model")
	if got := p.Name(); got != "anthropic" {
		t.Errorf("Name() = %q, want %q", got, "anthropic")
	}
}

// ---------------------------------------------------------------------------
// Test 2: Chat end_turn — text response
// ---------------------------------------------------------------------------

func TestAnthropicChat_EndTurn(t *testing.T) {
	apiResp := anthropicResponse{
		ID: "msg_001",
		Content: []anthropicContentBlock{
			{Type: "text", Text: "Hello, world!"},
		},
		StopReason: "end_turn",
	}
	apiResp.Usage.InputTokens = 10
	apiResp.Usage.OutputTokens = 5

	srv := mockServer(t, apiResp)
	defer srv.Close()

	p := newAnthropicTestProvider(srv.URL)
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
	if resp.PromptTok != 10 {
		t.Errorf("PromptTok = %d, want 10", resp.PromptTok)
	}
	if resp.OutputTok != 5 {
		t.Errorf("OutputTok = %d, want 5", resp.OutputTok)
	}
	if len(resp.ToolCalls) != 0 {
		t.Errorf("ToolCalls should be empty, got %d", len(resp.ToolCalls))
	}
}

// ---------------------------------------------------------------------------
// Test 3: Chat tool_use — ToolCalls populated
// ---------------------------------------------------------------------------

func TestAnthropicChat_ToolUse(t *testing.T) {
	apiResp := anthropicResponse{
		ID: "msg_002",
		Content: []anthropicContentBlock{
			{
				Type:  "tool_use",
				ID:    "tc_1",
				Name:  "read_file",
				Input: map[string]any{"path": "/tmp/test.txt"},
			},
		},
		StopReason: "tool_use",
	}
	apiResp.Usage.InputTokens = 20
	apiResp.Usage.OutputTokens = 8

	srv := mockServer(t, apiResp)
	defer srv.Close()

	p := newAnthropicTestProvider(srv.URL)
	resp, err := p.Chat([]Message{{Role: "user", Content: "Read that file"}}, []ToolDef{
		{
			Name:        "read_file",
			Description: "Read a file from disk",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
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
	if tc.ID != "tc_1" {
		t.Errorf("ToolCall.ID = %q, want %q", tc.ID, "tc_1")
	}
	if tc.Name != "read_file" {
		t.Errorf("ToolCall.Name = %q, want %q", tc.Name, "read_file")
	}
	if tc.Params["path"] != "/tmp/test.txt" {
		t.Errorf("ToolCall.Params[path] = %q, want %q", tc.Params["path"], "/tmp/test.txt")
	}
}

// ---------------------------------------------------------------------------
// Test 4: System prompt extraction
// ---------------------------------------------------------------------------

func TestAnthropicChat_SystemPrompt(t *testing.T) {
	apiResp := anthropicResponse{
		ID:         "msg_003",
		Content:    []anthropicContentBlock{{Type: "text", Text: "ok"}},
		StopReason: "end_turn",
	}

	var captured anthropicRequest
	srv := captureServer(t, &captured, apiResp)
	defer srv.Close()

	p := newAnthropicTestProvider(srv.URL)
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello"},
	}
	_, err := p.Chat(messages, nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	if captured.System != "You are a helpful assistant." {
		t.Errorf("System = %q, want %q", captured.System, "You are a helpful assistant.")
	}

	// The system message should NOT appear in the messages array.
	for _, m := range captured.Messages {
		if m.Role == "system" {
			t.Error("system role message found in messages array — should be in System field only")
		}
	}

	// The user message should still be present.
	if len(captured.Messages) != 1 {
		t.Errorf("len(Messages) = %d, want 1", len(captured.Messages))
	}
}

// ---------------------------------------------------------------------------
// Test 5: tool_result formatting
// ---------------------------------------------------------------------------

func TestAnthropicChat_ToolResult(t *testing.T) {
	apiResp := anthropicResponse{
		ID:         "msg_004",
		Content:    []anthropicContentBlock{{Type: "text", Text: "Done."}},
		StopReason: "end_turn",
	}

	var captured anthropicRequest
	srv := captureServer(t, &captured, apiResp)
	defer srv.Close()

	p := newAnthropicTestProvider(srv.URL)
	messages := []Message{
		{Role: "user", Content: "Read the file"},
		{Role: "assistant", Content: "Sure"},
		{
			Role:       "tool_result",
			Content:    "file contents here",
			ToolCallID: "tc_42",
		},
	}
	_, err := p.Chat(messages, nil)
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	// We expect 3 messages: user, assistant, user(tool_result).
	if len(captured.Messages) != 3 {
		t.Fatalf("len(Messages) = %d, want 3", len(captured.Messages))
	}

	// The last message must be role "user" (tool_result is wrapped as user).
	last := captured.Messages[2]
	if last.Role != "user" {
		t.Errorf("last message Role = %q, want %q", last.Role, "user")
	}

	// Decode the content blocks.
	var blocks []anthropicContentBlock
	if err := json.Unmarshal(last.Content, &blocks); err != nil {
		t.Fatalf("decode last message content: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("len(blocks) = %d, want 1", len(blocks))
	}
	block := blocks[0]
	if block.Type != "tool_result" {
		t.Errorf("block.Type = %q, want %q", block.Type, "tool_result")
	}
	if block.ToolUseID != "tc_42" {
		t.Errorf("block.ToolUseID = %q, want %q", block.ToolUseID, "tc_42")
	}
	if block.Content != "file contents here" {
		t.Errorf("block.Content = %q, want %q", block.Content, "file contents here")
	}
}

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

func TestAnthropicProviderImplementsProvider(t *testing.T) {
	var _ Provider = (*AnthropicProvider)(nil)
}
