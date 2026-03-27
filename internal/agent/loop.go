// Package agent implements the ShellForge agentic execution loop.
//
// Uses format-agnostic intent parsing: extracts action intent from ANY
// model output format (structured tool_calls, JSON blocks, XML tags,
// bare JSON, OpenAI function_call format). Every extracted action goes
// through AgentGuard governance — no exceptions.
//
// This is the core of ShellForge's moat: you cannot trust the transport
// layer for action integrity. The intent parser makes ShellForge
// model-agnostic and format-agnostic.
package agent

import (
"fmt"
"time"

"github.com/AgentGuardHQ/shellforge/internal/action"
"github.com/AgentGuardHQ/shellforge/internal/correction"
"github.com/AgentGuardHQ/shellforge/internal/governance"
"github.com/AgentGuardHQ/shellforge/internal/intent"
"github.com/AgentGuardHQ/shellforge/internal/logger"
"github.com/AgentGuardHQ/shellforge/internal/normalizer"
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

func RunLoop(cfg LoopConfig, engine *governance.Engine) (*RunResult, error) {
start := time.Now()
logger.Init(cfg.OutputDir, cfg.Agent)
defer logger.Close()

// Orchestrator integration: generate run identity and correction engine.
runID := fmt.Sprintf("run_%d", time.Now().UnixMilli())
var seq int
corrector := correction.NewEngine(3, 10) // 3 retries per action, 10 total budget

systemPrompt := buildSystemPrompt(cfg.System)
messages := []ollama.ChatMessage{
{Role: "system", Content: systemPrompt},
{Role: "user", Content: cfg.UserPrompt},
}

result := &RunResult{}
var log []string

logger.Agent(cfg.Agent, fmt.Sprintf("starting — max %d turns, model: %s, run: %s", cfg.MaxTurns, cfg.Model, runID))

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

// ── Intent parser: extract action from ANY format ──
// This is the format-agnostic layer. Works regardless of whether the model
// emits structured tool_calls, JSON blocks, XML tags, or bare JSON.
parsed := intent.Parse(content)
if parsed == nil {
// No actionable intent — this is a final answer.
result.Output = content
result.Turns = turn
logger.Agent(cfg.Agent, fmt.Sprintf("done — %d turns, %d tool calls", turn, result.ToolCalls))
break
}

logger.Agent(cfg.Agent, fmt.Sprintf("intent: %s via %s", parsed.Tool, parsed.Source))

result.ToolCalls++
seq++

// ── Normalizer: convert extracted intent to Canonical Action Representation ──
proposal := normalizer.Normalize(runID, seq, cfg.Agent, parsed.Tool, parsed.Params)
fp := normalizer.Fingerprint(proposal)

// ── Correction engine: check if this action should be retried or skipped ──
canAttempt, skipReason := corrector.ShouldCorrect(fp)
if !canAttempt {
// Too many retries or in lockdown — skip this action entirely.
logger.Agent(cfg.Agent, fmt.Sprintf("action skipped: %s", skipReason))
messages = append(messages, ollama.ChatMessage{
Role:    "user",
Content: fmt.Sprintf("Tool %q was skipped: %s. Try a different approach.", parsed.Tool, skipReason),
})
summary := fmt.Sprintf("[turn %d] %s → skipped (%s)", turn, parsed.Tool, skipReason)
log = append(log, summary)
continue
}

// ── Governance evaluation (existing) ──
decision := engine.Evaluate(parsed.Tool, parsed.Params)

if !decision.Allowed {
result.Denials++

// Map governance.Decision to action.GovernanceDecision for the correction engine.
govDecision := action.GovernanceDecision{
Allowed:  false,
Decision: "deny",
Reason:   decision.Reason,
Rule:     decision.PolicyName,
}

// Record denial and attempt correction.
corrector.RecordDenial(fp, govDecision)
logger.Governance(cfg.Agent, parsed.Tool, parsed.Params, decision.Allowed, decision.PolicyName, decision.Reason)

canCorrect, _ := corrector.ShouldCorrect(fp)
if canCorrect {
// Build corrective feedback and feed it back to the LLM.
feedback := corrector.BuildFeedback(proposal, govDecision)
logger.Agent(cfg.Agent, fmt.Sprintf("governance denied %q — sending correction feedback (escalation: %s)", parsed.Tool, corrector.Level()))
messages = append(messages, ollama.ChatMessage{
Role:    "user",
Content: feedback,
})
} else {
// Exhausted retries — skip and inform the LLM.
logger.Agent(cfg.Agent, fmt.Sprintf("governance denied %q — no retries left, skipping", parsed.Tool))
messages = append(messages, ollama.ChatMessage{
Role:    "user",
Content: fmt.Sprintf("Tool %q was denied and cannot be retried. Move on to a different approach.", parsed.Tool),
})
}

summary := fmt.Sprintf("[turn %d] %s → denied (%s)", turn, parsed.Tool, decision.PolicyName)
log = append(log, summary)
continue
}

// ── Governance allowed: log and execute tool ──
logger.Governance(cfg.Agent, parsed.Tool, parsed.Params, decision.Allowed, decision.PolicyName, decision.Reason)
toolResult := tools.ExecuteDirect(parsed.Tool, parsed.Params, engine.GetTimeout())
logger.ToolResult(cfg.Agent, parsed.Tool, toolResult.Success, toolResult.Output)

var msg string
if toolResult.Success {
msg = fmt.Sprintf("Tool %q returned:\n%s", parsed.Tool, toolResult.Output)
} else {
msg = fmt.Sprintf("Tool %q failed: %s", parsed.Tool, toolResult.Output)
}
messages = append(messages, ollama.ChatMessage{Role: "user", Content: msg})

summary := fmt.Sprintf("[turn %d] %s → %s", turn, parsed.Tool, boolStr(toolResult.Success, "ok", "fail"))
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

// Old parseToolCall/tryParse removed — replaced by intent.Parse()
// which handles all formats: JSON blocks, XML tags, bare JSON,
// OpenAI function_call, and tool name/param aliasing.

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
- If governance denies a tool call, do NOT retry the same action — try an alternative.
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
