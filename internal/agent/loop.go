// Package agent implements the ShellForge agentic execution loop.
//
// This is the native engine — a minimal fallback for when OpenCode or
// DeepAgents aren't installed. It uses structured prompting to give
// Ollama tool-use capabilities, with every tool call routed through
// the AgentGuard governance engine.
//
// When frameworks plug in, they replace this loop but use the same
// governance layer (internal/tools + internal/governance).
package agent

import (
"encoding/json"
"fmt"
"regexp"
"strings"
"time"

"github.com/AgentGuardHQ/shellforge/internal/governance"
"github.com/AgentGuardHQ/shellforge/internal/logger"
"github.com/AgentGuardHQ/shellforge/internal/ollama"
"github.com/AgentGuardHQ/shellforge/internal/tools"
)

type LoopConfig struct {
Agent       string
System      string
UserPrompt  string
Model       string
MaxTurns    int
TimeoutMs   int
OutputDir   string
TokenBudget int
}

type RunResult struct {
Success     bool
Output      string
Turns       int
ToolCalls   int
Denials     int
PromptTok   int
ResponseTok int
DurationMs  int64
Log         []string
}

var (
jsonBlockRe = regexp.MustCompile("(?s)```json\\s*\\n?\\s*(\\{.*?\"tool\".*?\\})\\s*\\n?\\s*```")
toolTagRe   = regexp.MustCompile("(?s)<tool>(.*?)</tool>")
bareJSONRe  = regexp.MustCompile(`\{[^{}]*"tool"\s*:\s*"[^"]+"\s*,\s*"params"\s*:\s*\{[^}]*\}[^{}]*\}`)
)

func RunLoop(cfg LoopConfig, engine *governance.Engine) (*RunResult, error) {
start := time.Now()
logger.Init(cfg.OutputDir, cfg.Agent)
defer logger.Close()

systemPrompt := buildSystemPrompt(cfg.System)
messages := []ollama.ChatMessage{
{Role: "system", Content: systemPrompt},
{Role: "user", Content: cfg.UserPrompt},
}

result := &RunResult{}
var log []string

logger.Agent(cfg.Agent, fmt.Sprintf("starting — max %d turns, model: %s", cfg.MaxTurns, cfg.Model))

for turn := 1; turn <= cfg.MaxTurns; turn++ {
elapsed := time.Since(start).Milliseconds()
if int(elapsed) > cfg.TimeoutMs {
logger.Agent(cfg.Agent, fmt.Sprintf("timeout after %d turns", turn-1))
break
}

// Compact if needed
compacted := compactMessages(messages, cfg.TokenBudget)

tokEst := estimateTokens(compacted)
logger.Agent(cfg.Agent, fmt.Sprintf("turn %d/%d (~%d tokens)", turn, cfg.MaxTurns, tokEst))

resp, err := ollama.Chat(compacted, cfg.Model)
if err != nil {
logger.Error(cfg.Agent, "ollama: "+err.Error())
result.Output = "Model error: " + err.Error()
break
}

result.PromptTok += resp.PromptEval
result.ResponseTok += resp.EvalCount
logger.ModelCall(cfg.Agent, resp.PromptEval, resp.EvalCount, resp.TotalDuration/1_000_000)

content := resp.Message.Content
messages = append(messages, ollama.ChatMessage{Role: "assistant", Content: content})

toolCall := parseToolCall(content)
if toolCall == nil {
// No tool call — final answer
result.Output = content
result.Turns = turn
logger.Agent(cfg.Agent, fmt.Sprintf("done — %d turns, %d tool calls", turn, result.ToolCalls))
break
}

result.ToolCalls++
toolResult := tools.Execute(engine, cfg.Agent, toolCall.Tool, toolCall.Params)

if !toolResult.Success && strings.HasPrefix(toolResult.Output, "DENIED") {
result.Denials++
}

var msg string
if toolResult.Success {
msg = fmt.Sprintf("Tool %q returned:\n%s", toolCall.Tool, toolResult.Output)
} else {
msg = fmt.Sprintf("Tool %q failed: %s", toolCall.Tool, toolResult.Output)
}
messages = append(messages, ollama.ChatMessage{Role: "user", Content: msg})

summary := fmt.Sprintf("[turn %d] %s → %s", turn, toolCall.Tool, boolStr(toolResult.Success, "ok", "fail"))
log = append(log, summary)

// Last turn: force final answer
if turn == cfg.MaxTurns {
messages = append(messages, ollama.ChatMessage{
Role:    "user",
Content: "You've used all turns. Give your final answer now.",
})
final, err := ollama.Chat(compactMessages(messages, cfg.TokenBudget), cfg.Model)
if err == nil {
result.Output = final.Message.Content
result.PromptTok += final.PromptEval
result.ResponseTok += final.EvalCount
}
}
}

result.DurationMs = time.Since(start).Milliseconds()
result.Success = result.Denials == 0 || result.ToolCalls > result.Denials
result.Log = log
return result, nil
}

type toolCallParsed struct {
Tool   string            `json:"tool"`
Params map[string]string `json:"params"`
}

func parseToolCall(content string) *toolCallParsed {
// Try ```json block
if m := jsonBlockRe.FindStringSubmatch(content); len(m) > 1 {
if tc := tryParse(m[1]); tc != nil {
return tc
}
}
// Try <tool> tags
if m := toolTagRe.FindStringSubmatch(content); len(m) > 1 {
if tc := tryParse(m[1]); tc != nil {
return tc
}
}
// Try bare JSON
if m := bareJSONRe.FindString(content); m != "" {
if tc := tryParse(m); tc != nil {
return tc
}
}
return nil
}

func tryParse(s string) *toolCallParsed {
var tc toolCallParsed
if err := json.Unmarshal([]byte(s), &tc); err != nil {
return nil
}
if tc.Tool != "" && tc.Params != nil {
return &tc
}
return nil
}

func buildSystemPrompt(base string) string {
toolDocs := tools.FormatForPrompt()
return base + `

## Tools

You have access to these tools. To use one, respond with a JSON code block:

` + "```json\n{\"tool\": \"tool_name\", \"params\": {\"param1\": \"value1\"}}\n```" + `

Available tools:

` + toolDocs + `
## Rules
- Use ONE tool per response. Wait for the result before calling another.
- When done, respond normally WITHOUT a tool call — that is your final answer.
- If governance denies a tool call, do NOT retry it.
- Keep tool usage focused and minimal.`
}

func compactMessages(msgs []ollama.ChatMessage, budget int) []ollama.ChatMessage {
if budget <= 0 {
budget = 3000
}
total := estimateTokens(msgs)
if total <= budget {
return msgs
}

// Keep system (0), first user (1), and last N messages
result := []ollama.ChatMessage{msgs[0], msgs[1]}
remaining := msgs[2:]

// Drop tool results from the middle until we fit
for total > budget && len(remaining) > 4 {
remaining = remaining[2:] // drop oldest assistant+tool pair
total = estimateTokens(append(result, remaining...))
}
return append(result, remaining...)
}

func estimateTokens(msgs []ollama.ChatMessage) int {
total := 0
for _, m := range msgs {
total += len(m.Content) / 4 // rough approximation
}
return total
}

func boolStr(b bool, t, f string) string {
if b {
return t
}
return f
}
