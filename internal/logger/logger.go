package logger

import (
"encoding/json"
"fmt"
"os"
"path/filepath"
"strings"
"time"
)

var toolDisplay = map[string]string{
"read_file":    "Read",
"write_file":   "Write",
"run_shell":    "Bash",
"list_files":   "ListDir",
"search_files": "Search",
}

type Entry struct {
Timestamp string            `json:"timestamp"`
Agent     string            `json:"agent"`
Event     string            `json:"event"`
Tool      string            `json:"tool,omitempty"`
Params    map[string]string `json:"params,omitempty"`
Decision  *DecisionLog      `json:"decision,omitempty"`
Message   string            `json:"message,omitempty"`
Tokens    *TokenLog         `json:"tokens,omitempty"`
Duration  int64             `json:"duration_ms,omitempty"`
}

type DecisionLog struct {
Allowed    bool   `json:"allowed"`
PolicyName string `json:"policy_name"`
Reason     string `json:"reason"`
}

type TokenLog struct {
Prompt   int `json:"prompt"`
Response int `json:"response"`
}

var (
entries []Entry
logFile *os.File
)

// Init opens a JSONL log file under outputDir named "<agent>-<timestamp>.jsonl".
// Must be called before any log functions; call Close when done.
func Init(outputDir, agent string) error {
if err := os.MkdirAll(outputDir, 0o755); err != nil {
return err
}
ts := time.Now().Format("2006-01-02T15-04-05")
path := filepath.Join(outputDir, fmt.Sprintf("%s-%s.jsonl", agent, ts))
f, err := os.Create(path)
if err != nil {
return err
}
logFile = f
return nil
}

// Close flushes and closes the current log file.
func Close() {
if logFile != nil {
logFile.Close()
}
}

func record(e Entry) {
entries = append(entries, e)
if logFile != nil {
data, _ := json.Marshal(e)
logFile.Write(data)
logFile.WriteString("\n")
}
}

// Governance logs a governance evaluation result to stdout and the JSONL log.
func Governance(agent, tool string, params map[string]string, allowed bool, policyName, reason string) {
status := "allow"
if !allowed {
status = "DENY"
}
display := toolDisplay[tool]
if display == "" {
display = tool
}
paramSummary := summarize(params)
fmt.Printf("[🛡️ Chitin] policy: %s — %s → %s(%s)\n", status, agent, display, paramSummary)
if !allowed {
fmt.Printf("  ↳ %s: %s\n", policyName, reason)
}

record(Entry{
Timestamp: time.Now().UTC().Format(time.RFC3339),
Agent:     agent,
Event:     "governance",
Tool:      tool,
Params:    params,
Decision:  &DecisionLog{Allowed: allowed, PolicyName: policyName, Reason: reason},
})
}

// ToolResult logs the outcome of a tool execution to stdout and the JSONL log.
func ToolResult(agent, tool string, success bool, output string) {
icon := "✓"
if !success {
icon = "✗"
}
display := toolDisplay[tool]
if display == "" {
display = tool
}
preview := strings.SplitN(output, "\n", 2)[0]
if len(preview) > 80 {
preview = preview[:77] + "..."
}
fmt.Printf("  %s %s → %s\n", icon, display, preview)

record(Entry{
Timestamp: time.Now().UTC().Format(time.RFC3339),
Agent:     agent,
Event:     "tool_call",
Tool:      tool,
Message:   truncate(output, 200),
})
}

// Agent logs a free-form info message from the named agent.
func Agent(agent, message string) {
fmt.Printf("[%s] %s\n", agent, message)
record(Entry{
Timestamp: time.Now().UTC().Format(time.RFC3339),
Agent:     agent,
Event:     "info",
Message:   message,
})
}

// ModelCall logs token usage and latency for an Ollama inference call.
func ModelCall(agent string, promptTokens, responseTokens int, durationMs int64) {
record(Entry{
Timestamp: time.Now().UTC().Format(time.RFC3339),
Agent:     agent,
Event:     "model_call",
Tokens:    &TokenLog{Prompt: promptTokens, Response: responseTokens},
Duration:  durationMs,
})
}

// Error logs an error message to stderr and the JSONL log.
func Error(agent, message string) {
fmt.Fprintf(os.Stderr, "[%s] ERROR: %s\n", agent, message)
record(Entry{
Timestamp: time.Now().UTC().Format(time.RFC3339),
Agent:     agent,
Event:     "error",
Message:   message,
})
}

// GetEntries returns all log entries recorded in this session (in-memory only).
func GetEntries() []Entry { return entries }

func summarize(params map[string]string) string {
parts := make([]string, 0, len(params))
for _, v := range params {
parts = append(parts, v)
}
s := strings.Join(parts, ", ")
if len(s) > 60 {
return s[:57] + "..."
}
return s
}

func truncate(s string, max int) string {
if len(s) <= max {
return s
}
return s[:max-3] + "..."
}
