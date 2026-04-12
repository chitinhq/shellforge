package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// NemoClawEngine is a thin wrapper around OpenClaw that additionally:
//   - Defaults to Nemotron as the model (instead of Claude)
//   - Enables OpenShell sandbox (if available)
//   - Sets additional security flags for hardened execution
//
// NemoClaw is an optional adapter — it never introduces a hard dependency.
type NemoClawEngine struct{}

func (e *NemoClawEngine) Name() string { return "nemoclaw" }

func (e *NemoClawEngine) Available() bool {
	// NemoClaw requires OpenClaw to be installed
	oc := &OpenClawEngine{}
	return oc.Available()
}

func (e *NemoClawEngine) Run(task Task) (*Result, error) {
	start := time.Now()

	if !e.Available() {
		return nil, fmt.Errorf("nemoclaw requires openclaw. Install: npm i -g @anthropic-ai/openclaw")
	}

	// Determine binary: standalone or npx (reuse OpenClaw resolution)
	binary, args := openclawCommand()

	// NemoClaw defaults to Nemotron model
	model := "nemotron"
	if task.Model != "" {
		model = task.Model
	}
	if v := os.Getenv("NEMOCLAW_MODEL"); v != "" {
		model = v
	}
	args = append(args, "--model", model)

	// Always headless for NemoClaw (security-oriented, no browser UI)
	args = append(args, "--headless")

	// Enable sandbox mode if OpenShell is available
	if _, err := exec.LookPath("openshell"); err == nil {
		args = append(args, "--sandbox", "openshell")
	}

	// Additional security flags
	args = append(args, "--no-network-write")
	args = append(args, "--read-only-fs")

	// Timeout
	if task.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", task.Timeout))
	}

	// Max turns
	if task.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", task.MaxTurns))
	}

	// Non-interactive for headless execution
	args = append(args, "--non-interactive")

	// Set up Chitin governance + NemoClaw-specific env
	env := append(os.Environ(),
		"CHITIN_POLICY=chitin.yaml",
		"CHITIN_MODE=enforce",
		"NEMOCLAW_SANDBOX=1",
	)

	// Wrap shell commands through govern-shell.sh
	governShell := findGovernShell()
	if governShell != "" {
		env = append(env,
			"SHELL="+governShell,
			"SHELLFORGE_REAL_SHELL=/bin/bash",
		)
	}

	cmd := exec.Command(binary, args...)
	cmd.Dir = task.WorkDir
	cmd.Env = env
	cmd.Stdin = strings.NewReader(task.Prompt)

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	duration := time.Since(start).Milliseconds()

	if err != nil && output == "" {
		return nil, fmt.Errorf("nemoclaw failed: %w", err)
	}

	// Parse structured output if available
	result := &Result{
		Success:    err == nil,
		Output:     output,
		DurationMs: duration,
	}

	// Try to extract metrics from OpenClaw's JSON output
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
