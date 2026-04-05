package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chitinhq/shellforge/internal/governance"
	"github.com/chitinhq/shellforge/internal/llm"
)

// mockProvider is a test double that returns pre-configured responses.
type mockProvider struct {
	name      string
	responses []*llm.Response
	calls     int
	received  []mockCall
}

type mockCall struct {
	Messages []llm.Message
	Tools    []llm.ToolDef
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) Chat(messages []llm.Message, tools []llm.ToolDef) (*llm.Response, error) {
	m.received = append(m.received, mockCall{Messages: messages, Tools: tools})
	if m.calls >= len(m.responses) {
		return nil, fmt.Errorf("mock: no more responses (called %d times, have %d)", m.calls+1, len(m.responses))
	}
	resp := m.responses[m.calls]
	m.calls++
	return resp, nil
}

// newPermissiveEngine creates a governance engine that allows everything.
func newPermissiveEngine(t *testing.T) *governance.Engine {
	t.Helper()
	return &governance.Engine{
		Mode:     "enforce",
		Policies: nil, // no policies = default-allow
	}
}

// newDenyShellEngine creates a governance engine that denies run_shell commands
// containing "ls".
func newDenyShellEngine(t *testing.T) *governance.Engine {
	t.Helper()
	return &governance.Engine{
		Mode: "enforce",
		Policies: []governance.Policy{
			{
				Name:        "deny-shell",
				Description: "deny ls shell commands",
				Match:       governance.Match{Command: "ls"},
				Action:      "deny",
				Message:     "shell commands are not allowed",
			},
		},
	}
}

// baseCfg returns a LoopConfig with reasonable test defaults.
func baseCfg(provider llm.Provider, outputDir string) LoopConfig {
	return LoopConfig{
		Agent:       "test-agent",
		System:      "You are a test assistant.",
		UserPrompt:  "What files are in this directory?",
		Model:       "test-model",
		MaxTurns:    10,
		TimeoutMs:   30000,
		OutputDir:   outputDir,
		TokenBudget: 8000,
		Provider:    provider,
	}
}

// TestProviderToolCallThenFinalAnswer verifies that the loop processes a
// single tool call via native tool-use and then accepts a final answer.
func TestProviderToolCallThenFinalAnswer(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file for list_files to find.
	testFile := filepath.Join(tmpDir, "hello.txt")
	os.WriteFile(testFile, []byte("hello world"), 0644)

	mock := &mockProvider{
		name: "mock-anthropic",
		responses: []*llm.Response{
			// Turn 1: model requests list_files tool
			{
				Content:    "",
				StopReason: "tool_use",
				ToolCalls: []llm.ToolCall{
					{
						ID:     "call_001",
						Name:   "list_files",
						Params: map[string]string{"directory": tmpDir},
					},
				},
				PromptTok: 100,
				OutputTok: 20,
			},
			// Turn 2: model gives final answer
			{
				Content:    "The directory contains hello.txt.",
				StopReason: "end_turn",
				ToolCalls:  nil,
				PromptTok:  150,
				OutputTok:  30,
			},
		},
	}

	cfg := baseCfg(mock, tmpDir)
	engine := newPermissiveEngine(t)

	result, err := RunLoop(cfg, engine)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	if result.ExitReason != "final_answer" {
		t.Errorf("ExitReason: got %q, want %q", result.ExitReason, "final_answer")
	}
	if !result.Success {
		t.Errorf("Success: got false, want true")
	}
	if result.ToolCalls != 1 {
		t.Errorf("ToolCalls: got %d, want %d", result.ToolCalls, 1)
	}
	if result.Denials != 0 {
		t.Errorf("Denials: got %d, want %d", result.Denials, 0)
	}
	if result.Output != "The directory contains hello.txt." {
		t.Errorf("Output: got %q, want %q", result.Output, "The directory contains hello.txt.")
	}
	if result.Turns != 2 {
		t.Errorf("Turns: got %d, want %d", result.Turns, 2)
	}
	if result.PromptTok != 250 {
		t.Errorf("PromptTok: got %d, want %d", result.PromptTok, 250)
	}
	if result.ResponseTok != 50 {
		t.Errorf("ResponseTok: got %d, want %d", result.ResponseTok, 50)
	}

	// Verify that tool definitions were passed to the provider.
	if len(mock.received) < 1 {
		t.Fatal("mock received no calls")
	}
	firstCall := mock.received[0]
	if len(firstCall.Tools) == 0 {
		t.Error("first call should have received tool definitions")
	}

	// Verify that the second call has a tool_result message.
	if len(mock.received) < 2 {
		t.Fatal("mock should have received 2 calls")
	}
	secondCallMsgs := mock.received[1].Messages
	hasToolResult := false
	for _, m := range secondCallMsgs {
		if m.Role == "tool_result" && m.ToolCallID == "call_001" {
			hasToolResult = true
			break
		}
	}
	if !hasToolResult {
		t.Error("second call should contain a tool_result message with ToolCallID=call_001")
	}
}

// TestProviderGovernanceDenial verifies that governance denials on native
// tool calls result in a tool_result message with denial feedback.
func TestProviderGovernanceDenial(t *testing.T) {
	tmpDir := t.TempDir()

	mock := &mockProvider{
		name: "mock-anthropic",
		responses: []*llm.Response{
			// Turn 1: model requests run_shell (will be denied)
			{
				Content:    "",
				StopReason: "tool_use",
				ToolCalls: []llm.ToolCall{
					{
						ID:     "call_shell_1",
						Name:   "run_shell",
						Params: map[string]string{"command": "ls -la"},
					},
				},
				PromptTok: 100,
				OutputTok: 20,
			},
			// Turn 2: model gives final answer after denial
			{
				Content:    "I cannot run shell commands. Here is what I know.",
				StopReason: "end_turn",
				ToolCalls:  nil,
				PromptTok:  200,
				OutputTok:  40,
			},
		},
	}

	cfg := baseCfg(mock, tmpDir)
	engine := newDenyShellEngine(t)

	result, err := RunLoop(cfg, engine)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	if result.ExitReason != "final_answer" {
		t.Errorf("ExitReason: got %q, want %q", result.ExitReason, "final_answer")
	}
	if result.Denials != 1 {
		t.Errorf("Denials: got %d, want %d", result.Denials, 1)
	}
	if result.ToolCalls != 1 {
		t.Errorf("ToolCalls: got %d, want %d", result.ToolCalls, 1)
	}
	if result.Output != "I cannot run shell commands. Here is what I know." {
		t.Errorf("Output: got %q", result.Output)
	}

	// The second call should have a tool_result message with denial feedback.
	if len(mock.received) < 2 {
		t.Fatal("mock should have received 2 calls")
	}
	secondCallMsgs := mock.received[1].Messages
	hasToolResult := false
	for _, m := range secondCallMsgs {
		if m.Role == "tool_result" && m.ToolCallID == "call_shell_1" {
			hasToolResult = true
			break
		}
	}
	if !hasToolResult {
		t.Error("second call should contain a tool_result message for denied tool call")
	}
}

// TestProviderNoToolCalls verifies that when the provider returns
// immediately with no tool calls (end_turn), the loop exits as final_answer.
func TestProviderNoToolCalls(t *testing.T) {
	tmpDir := t.TempDir()

	mock := &mockProvider{
		name: "mock-anthropic",
		responses: []*llm.Response{
			{
				Content:    "I can answer directly: this is a test.",
				StopReason: "end_turn",
				ToolCalls:  nil,
				PromptTok:  80,
				OutputTok:  25,
			},
		},
	}

	cfg := baseCfg(mock, tmpDir)
	engine := newPermissiveEngine(t)

	result, err := RunLoop(cfg, engine)
	if err != nil {
		t.Fatalf("RunLoop error: %v", err)
	}

	if result.ExitReason != "final_answer" {
		t.Errorf("ExitReason: got %q, want %q", result.ExitReason, "final_answer")
	}
	if !result.Success {
		t.Errorf("Success: got false, want true")
	}
	if result.ToolCalls != 0 {
		t.Errorf("ToolCalls: got %d, want %d", result.ToolCalls, 0)
	}
	if result.Denials != 0 {
		t.Errorf("Denials: got %d, want %d", result.Denials, 0)
	}
	if result.Turns != 1 {
		t.Errorf("Turns: got %d, want %d", result.Turns, 1)
	}
	if result.Output != "I can answer directly: this is a test." {
		t.Errorf("Output: got %q", result.Output)
	}
	if result.PromptTok != 80 {
		t.Errorf("PromptTok: got %d, want %d", result.PromptTok, 80)
	}

	// Only one call to the provider.
	if mock.calls != 1 {
		t.Errorf("provider calls: got %d, want %d", mock.calls, 1)
	}
}

// TestBuildToolDefs verifies that buildToolDefs produces valid llm.ToolDef
// entries from tools.Definitions.
func TestBuildToolDefs(t *testing.T) {
	defs := buildToolDefs()
	if len(defs) == 0 {
		t.Fatal("buildToolDefs returned empty slice")
	}

	// Check that each definition has a name, description, and parameters.
	for _, d := range defs {
		if d.Name == "" {
			t.Error("ToolDef has empty Name")
		}
		if d.Description == "" {
			t.Errorf("ToolDef %q has empty Description", d.Name)
		}
		if d.Parameters == nil {
			t.Errorf("ToolDef %q has nil Parameters", d.Name)
			continue
		}
		typ, ok := d.Parameters["type"]
		if !ok || typ != "object" {
			t.Errorf("ToolDef %q: Parameters[type] = %v, want %q", d.Name, typ, "object")
		}
		props, ok := d.Parameters["properties"]
		if !ok || props == nil {
			t.Errorf("ToolDef %q: missing properties", d.Name)
		}
	}

	// Spot-check read_file has a "path" property.
	found := false
	for _, d := range defs {
		if d.Name == "read_file" {
			found = true
			props := d.Parameters["properties"].(map[string]any)
			if _, ok := props["path"]; !ok {
				t.Error("read_file ToolDef missing 'path' property")
			}
			req := d.Parameters["required"].([]string)
			if len(req) != 1 || req[0] != "path" {
				t.Errorf("read_file required: got %v, want [path]", req)
			}
		}
	}
	if !found {
		t.Error("buildToolDefs missing read_file definition")
	}
}
