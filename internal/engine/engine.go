// Package engine defines the pluggable engine interface.
// ShellForge's value is governance — the engine provides the agentic loop.
// Native is the fallback. OpenCode and DeepAgents are the real engines.
package engine

// Engine runs an agentic task and returns the result.
type Engine interface {
Name() string
Available() bool
Run(task Task) (*Result, error)
}

type Task struct {
Agent   string
Prompt  string
System  string
WorkDir string
Model   string
MaxTurns int
Timeout  int // seconds
}

type Result struct {
Success     bool
Output      string
Turns       int
ToolCalls   int
Denials     int
PromptTok   int
ResponseTok int
DurationMs  int64
}
