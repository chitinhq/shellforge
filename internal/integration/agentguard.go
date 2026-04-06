package integration

import (
"encoding/json"
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"
)

// AgentGuardKernel — the real AgentGuard Go kernel integration.
// When the full kernel is installed, ShellForge delegates policy evaluation
// to the production-grade engine with blast radius analysis, persona-aware
// decisions, invariant checking, and sub-millisecond evaluation.
//
// The kernel lives at: github.com/chitinhq/chitin/go
// Install: go install github.com/chitinhq/chitin/go/cmd/agentguard@latest
//
// Without the kernel, ShellForge uses its built-in YAML evaluator (internal/governance).
// With the kernel, we get: deny/allow/escalate/intervene decisions, corrected commands,
// blast radius scoring, and structured audit events.
type AgentGuardKernel struct {
enabled bool
binPath string
}

func NewAgentGuardKernel() *AgentGuardKernel {
// Check for installed agentguard binary
path, err := exec.LookPath("agentguard")
if err != nil {
// Check workspace location
workspace := os.Getenv("AGENTGUARD_WORKSPACE")
if workspace == "" {
workspace = filepath.Join(os.Getenv("HOME"), "agentguard-workspace")
}
candidate := filepath.Join(workspace, "agent-guard", "go", "agentguard")
if _, err := os.Stat(candidate); err == nil {
return &AgentGuardKernel{enabled: true, binPath: candidate}
}
return &AgentGuardKernel{enabled: false}
}
return &AgentGuardKernel{enabled: true, binPath: path}
}

func (k *AgentGuardKernel) Available() bool { return k.enabled }
func (k *AgentGuardKernel) Name() string    { return "agentguard-kernel" }

// HookInput matches the kernel's pkg/hook.HookInput format.
type HookInput struct {
Tool      string          `json:"tool"`
Input     json.RawMessage `json:"input"`
SessionID string          `json:"sessionId"`
Event     string          `json:"event"` // "preToolUse" | "postToolUse"
}

// HookResponse from the kernel.
type HookResponse struct {
Decision         string `json:"decision"`         // allow | deny
Reason           string `json:"reason"`
Suggestion       string `json:"suggestion,omitempty"`
CorrectedCommand string `json:"correctedCommand,omitempty"`
}

// Evaluate runs a tool call through the full AgentGuard kernel.
// This gives us: blast radius analysis, persona-aware decisions,
// invariant checking, and corrected command suggestions.
func (k *AgentGuardKernel) Evaluate(tool string, params map[string]string) (*HookResponse, error) {
if !k.enabled {
return nil, fmt.Errorf("agentguard kernel not installed")
}

input := HookInput{
Tool:      mapToolName(tool),
Input:     marshalInput(tool, params),
SessionID: "shellforge-" + fmt.Sprintf("%d", os.Getpid()),
Event:     "preToolUse",
}

inputJSON, _ := json.Marshal(input)

// The kernel reads from AGENTGUARD_HOOK_INPUT env var (Copilot mode)
// or stdin (Claude Code mode). We use env var.
cmd := exec.Command(k.binPath, "hook", "copilot")
cmd.Env = append(os.Environ(),
"AGENTGUARD_HOOK_INPUT="+string(inputJSON),
)
out, err := cmd.Output()
if err != nil {
return nil, fmt.Errorf("kernel evaluation failed: %w", err)
}

var resp HookResponse
if err := json.Unmarshal(out, &resp); err != nil {
return nil, fmt.Errorf("parse kernel response: %w", err)
}
return &resp, nil
}

func mapToolName(tool string) string {
switch tool {
case "run_shell":
return "Bash"
case "read_file":
return "Read"
case "write_file":
return "Write"
case "list_files":
return "ListDir"
case "search_files":
return "Search"
default:
return tool
}
}

func marshalInput(tool string, params map[string]string) json.RawMessage {
switch tool {
case "run_shell":
data, _ := json.Marshal(map[string]string{"command": params["command"]})
return data
case "write_file":
data, _ := json.Marshal(map[string]string{
"file_path": params["path"],
"content":   params["content"],
})
return data
case "read_file":
data, _ := json.Marshal(map[string]string{"file_path": params["path"]})
return data
default:
data, _ := json.Marshal(params)
return data
}
}

// Version returns the kernel version.
func (k *AgentGuardKernel) Version() string {
if !k.enabled {
return "not installed"
}
out, err := exec.Command(k.binPath, "--version").Output()
if err != nil {
return "unknown"
}
return strings.TrimSpace(string(out))
}
