package engine

import (
"encoding/json"
"fmt"
"os"
"os/exec"
"strings"
"time"
)

// OpenCodeEngine wraps the opencode-ai CLI with AgentGuard governance.
// OpenCode is a Go-native AI coding agent with built-in tool use.
// We run it as a subprocess and capture its structured output.
type OpenCodeEngine struct{}

func (e *OpenCodeEngine) Name() string { return "opencode" }

func (e *OpenCodeEngine) Available() bool {
_, err := exec.LookPath("opencode")
return err == nil
}

func (e *OpenCodeEngine) Run(task Task) (*Result, error) {
start := time.Now()

if !e.Available() {
return nil, fmt.Errorf("opencode not installed. Install: npm i -g opencode-ai")
}

// Build OpenCode command with governance hooks
// OpenCode supports --non-interactive for headless mode
args := []string{
"--non-interactive",
"--model", task.Model,
}

// Set up AgentGuard as an OpenCode plugin via env
env := append(os.Environ(),
"AGENTGUARD_POLICY=agentguard.yaml",
"AGENTGUARD_MODE=enforce",
fmt.Sprintf("OPENCODE_TIMEOUT=%d", task.Timeout),
)

cmd := exec.Command("opencode", args...)
cmd.Dir = task.WorkDir
cmd.Env = env
cmd.Stdin = strings.NewReader(task.Prompt)

out, err := cmd.CombinedOutput()
output := strings.TrimSpace(string(out))

duration := time.Since(start).Milliseconds()

if err != nil && output == "" {
return nil, fmt.Errorf("opencode failed: %w", err)
}

// Parse structured output if available
result := &Result{
Success:    err == nil,
Output:     output,
DurationMs: duration,
}

// Try to extract metrics from OpenCode's JSON output
if idx := strings.LastIndex(output, "\n{"); idx >= 0 {
var metrics struct {
Turns     int `json:"turns"`
ToolCalls int `json:"tool_calls"`
Tokens    struct {
Prompt   int `json:"prompt"`
Response int `json:"response"`
} `json:"tokens"`
}
if json.Unmarshal([]byte(output[idx+1:]), &metrics) == nil {
result.Turns = metrics.Turns
result.ToolCalls = metrics.ToolCalls
result.PromptTok = metrics.Tokens.Prompt
result.ResponseTok = metrics.Tokens.Response
result.Output = strings.TrimSpace(output[:idx])
}
}

return result, nil
}
