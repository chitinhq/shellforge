package llm

import (
	"testing"
)

func TestMessageFields(t *testing.T) {
	m := Message{
		Role:       "tool_result",
		Content:    "output here",
		ToolCallID: "call_abc123",
	}
	if m.Role != "tool_result" {
		t.Errorf("Role: got %q, want %q", m.Role, "tool_result")
	}
	if m.Content != "output here" {
		t.Errorf("Content: got %q, want %q", m.Content, "output here")
	}
	if m.ToolCallID != "call_abc123" {
		t.Errorf("ToolCallID: got %q, want %q", m.ToolCallID, "call_abc123")
	}
}

func TestToolDefFields(t *testing.T) {
	td := ToolDef{
		Name:        "read_file",
		Description: "Reads a file from disk",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
	}
	if td.Name != "read_file" {
		t.Errorf("Name: got %q, want %q", td.Name, "read_file")
	}
	if td.Description == "" {
		t.Error("Description should not be empty")
	}
	if td.Parameters == nil {
		t.Error("Parameters should not be nil")
	}
}

func TestToolCallFields(t *testing.T) {
	tc := ToolCall{
		ID:     "call_1",
		Name:   "list_files",
		Params: map[string]string{"directory": "."},
	}
	if tc.ID != "call_1" {
		t.Errorf("ID: got %q, want %q", tc.ID, "call_1")
	}
	if tc.Name != "list_files" {
		t.Errorf("Name: got %q, want %q", tc.Name, "list_files")
	}
	if tc.Params["directory"] != "." {
		t.Errorf("Params[directory]: got %q, want %q", tc.Params["directory"], ".")
	}
}

func TestResponseFields(t *testing.T) {
	r := Response{
		Content:    "here is the answer",
		ToolCalls:  nil,
		StopReason: "end_turn",
		PromptTok:  100,
		OutputTok:  50,
	}
	if r.Content != "here is the answer" {
		t.Errorf("Content: got %q, want %q", r.Content, "here is the answer")
	}
	if r.StopReason != "end_turn" {
		t.Errorf("StopReason: got %q, want %q", r.StopReason, "end_turn")
	}
	if r.PromptTok != 100 {
		t.Errorf("PromptTok: got %d, want %d", r.PromptTok, 100)
	}
	if r.OutputTok != 50 {
		t.Errorf("OutputTok: got %d, want %d", r.OutputTok, 50)
	}
	if r.ToolCalls != nil {
		t.Error("ToolCalls should be nil for end_turn")
	}
}

func TestOllamaProviderName(t *testing.T) {
	p := NewOllamaProvider("http://localhost:11434", "qwen3:1.7b")
	if p.Name() != "ollama" {
		t.Errorf("Name: got %q, want %q", p.Name(), "ollama")
	}
}

func TestOllamaProviderNameEmptyArgs(t *testing.T) {
	p := NewOllamaProvider("", "")
	if p.Name() != "ollama" {
		t.Errorf("Name: got %q, want %q", p.Name(), "ollama")
	}
}

func TestOllamaProviderImplementsProvider(t *testing.T) {
	// Compile-time check: *OllamaProvider must satisfy Provider interface.
	var _ Provider = (*OllamaProvider)(nil)
}
