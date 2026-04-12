package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// OpenClawEngine wraps the OpenClaw CLI with Chitin governance.
// OpenClaw is a browser automation + integrations runtime by Anthropic.
// Installed via npm: npx @anthropic-ai/openclaw, or as a standalone binary.
// Supports headless mode (for server/RunPod) and Extension Relay mode (local Mac with browser).
type OpenClawEngine struct{}

func (e *OpenClawEngine) Name() string { return "openclaw" }

func (e *OpenClawEngine) Available() bool {
	// Check for standalone binary first
	if _, err := exec.LookPath("openclaw"); err == nil {
		return true
	}
	// Check for npx availability (npm-installed)
	cmd := exec.Command("npx", "@anthropic-ai/openclaw", "--version")
	return cmd.Run() == nil
}

func (e *OpenClawEngine) Run(task Task) (*Result, error) {
	start := time.Now()

	if !e.Available() {
		return nil, fmt.Errorf("openclaw not installed. Install: npm i -g @anthropic-ai/openclaw")
	}

	// Determine binary: standalone or npx
	binary, args := openclawCommand()

	// Headless mode: default on for server (no DISPLAY), override via env
	headless := os.Getenv("DISPLAY") == ""
	if v := os.Getenv("OPENCLAW_HEADLESS"); v != "" {
		headless = v == "1" || v == "true"
	}

	if headless {
		args = append(args, "--headless")
	}

	// Model override
	model := task.Model
	if v := os.Getenv("OPENCLAW_MODEL"); v != "" {
		model = v
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	// Timeout
	if task.Timeout > 0 {
		args = append(args, "--timeout", fmt.Sprintf("%d", task.Timeout))
	}

	// Max turns
	if task.MaxTurns > 0 {
		args = append(args, "--max-turns", fmt.Sprintf("%d", task.MaxTurns))
	}

	// Non-interactive mode for headless execution
	args = append(args, "--non-interactive")

	// Set up Chitin governance via env
	env := append(os.Environ(),
		"CHITIN_POLICY=chitin.yaml",
		"CHITIN_MODE=enforce",
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
		return nil, fmt.Errorf("openclaw failed: %w", err)
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

// openclawCommand returns the binary and base args for invoking OpenClaw.
// Prefers the standalone binary; falls back to npx.
func openclawCommand() (string, []string) {
	if _, err := exec.LookPath("openclaw"); err == nil {
		return "openclaw", nil
	}
	return "npx", []string{"@anthropic-ai/openclaw"}
}

// findGovernShell locates the govern-shell.sh wrapper for governance integration.
func findGovernShell() string {
	sfBin, _ := exec.LookPath("shellforge")
	if sfBin == "" {
		// Try common locations even without shellforge on PATH
		for _, path := range []string{
			"scripts/govern-shell.sh",
			"/usr/local/share/shellforge/govern-shell.sh",
		} {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		return ""
	}
	for _, path := range []string{
		filepath.Join(filepath.Dir(sfBin), "..", "share", "shellforge", "govern-shell.sh"),
		filepath.Join(filepath.Dir(sfBin), "govern-shell.sh"),
		"scripts/govern-shell.sh",
		"/usr/local/share/shellforge/govern-shell.sh",
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
