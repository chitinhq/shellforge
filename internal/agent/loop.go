// Package agent implements the ShellForge agentic execution loop.
//
// Uses format-agnostic intent parsing: extracts action intent from ANY
// model output format (structured tool_calls, JSON blocks, XML tags,
// bare JSON, OpenAI function_call format). Every extracted action goes
// through AgentGuard governance — no exceptions.
//
// When a Provider with native tool-use is set (e.g. Anthropic), the loop
// uses structured ToolCalls directly instead of text-based intent parsing.
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
	"github.com/AgentGuardHQ/shellforge/internal/llm"
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
	Provider    llm.Provider // optional; nil falls back to legacy ollama.Chat()
}

type RunResult struct {
	Success     bool
	ExitReason  string // "final_answer", "timeout", "max_turns", "model_error"
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

	if cfg.Provider != nil {
		return runProviderLoop(cfg, engine, start)
	}
	return runOllamaLoop(cfg, engine, start)
}

// ----------------------------------------------------------------------------
// Provider path: uses []llm.Message + native tool-use
// ----------------------------------------------------------------------------

func runProviderLoop(cfg LoopConfig, engine *governance.Engine, start time.Time) (*RunResult, error) {
	runID := fmt.Sprintf("run_%d", time.Now().UnixMilli())
	var seq int
	corrector := correction.NewEngine(3, 10)

	systemPrompt := buildSystemPrompt(cfg.System)
	messages := []llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: cfg.UserPrompt},
	}

	toolDefs := buildToolDefs()

	result := &RunResult{}
	var log []string

	logger.Agent(cfg.Agent, fmt.Sprintf("starting — max %d turns, provider: %s, run: %s", cfg.MaxTurns, cfg.Provider.Name(), runID))

	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		elapsed := time.Since(start).Milliseconds()
		if int(elapsed) > cfg.TimeoutMs {
			logger.Agent(cfg.Agent, fmt.Sprintf("timeout after %d turns", turn-1))
			result.Turns = turn - 1
			result.ExitReason = "timeout"
			break
		}

		compacted := compactLLMMessages(messages, cfg.TokenBudget)

		tokEst := estimateLLMTokens(compacted)
		logger.Agent(cfg.Agent, fmt.Sprintf("turn %d/%d (~%d tokens)", turn, cfg.MaxTurns, tokEst))

		provResp, perr := cfg.Provider.Chat(compacted, toolDefs)
		if perr != nil {
			logger.Error(cfg.Agent, cfg.Provider.Name()+": "+perr.Error())
			result.Output = "Model error: " + perr.Error()
			result.Turns = turn
			result.ExitReason = "model_error"
			break
		}

		result.PromptTok += provResp.PromptTok
		result.ResponseTok += provResp.OutputTok
		logger.ModelCall(cfg.Agent, provResp.PromptTok, provResp.OutputTok, 0)

		// ── Native tool-use path ──
		if len(provResp.ToolCalls) > 0 {
			// Append the assistant message (may contain text + tool_use blocks).
			messages = append(messages, llm.Message{Role: "assistant", Content: provResp.Content})

			for _, tc := range provResp.ToolCalls {
				logger.Agent(cfg.Agent, fmt.Sprintf("tool_use: %s (id: %s)", tc.Name, tc.ID))

				result.ToolCalls++
				seq++

				// ── Normalizer: convert to Canonical Action Representation ──
				proposal := normalizer.Normalize(runID, seq, cfg.Agent, tc.Name, tc.Params)
				fp := normalizer.Fingerprint(proposal)

				// ── Correction engine: check retry budget ──
				canAttempt, skipReason := corrector.ShouldCorrect(fp)
				if !canAttempt {
					logger.Agent(cfg.Agent, fmt.Sprintf("action skipped: %s", skipReason))
					messages = append(messages, llm.Message{
						Role:       "tool_result",
						Content:    fmt.Sprintf("Tool %q was skipped: %s. Try a different approach.", tc.Name, skipReason),
						ToolCallID: tc.ID,
					})
					log = append(log, fmt.Sprintf("[turn %d] %s → skipped (%s)", turn, tc.Name, skipReason))
					continue
				}

				// ── Governance evaluation ──
				decision := engine.Evaluate(tc.Name, tc.Params)

				if !decision.Allowed {
					result.Denials++

					govDecision := action.GovernanceDecision{
						Allowed:  false,
						Decision: "deny",
						Reason:   decision.Reason,
						Rule:     decision.PolicyName,
					}

					corrector.RecordDenial(fp, govDecision)
					logger.Governance(cfg.Agent, tc.Name, tc.Params, decision.Allowed, decision.PolicyName, decision.Reason)

					canCorrect, _ := corrector.ShouldCorrect(fp)
					var feedback string
					if canCorrect {
						feedback = corrector.BuildFeedback(proposal, govDecision)
						logger.Agent(cfg.Agent, fmt.Sprintf("governance denied %q — sending correction feedback (escalation: %s)", tc.Name, corrector.Level()))
					} else {
						feedback = fmt.Sprintf("Tool %q was denied and cannot be retried. Move on to a different approach.", tc.Name)
						logger.Agent(cfg.Agent, fmt.Sprintf("governance denied %q — no retries left, skipping", tc.Name))
					}

					messages = append(messages, llm.Message{
						Role:       "tool_result",
						Content:    feedback,
						ToolCallID: tc.ID,
					})
					log = append(log, fmt.Sprintf("[turn %d] %s → denied (%s)", turn, tc.Name, decision.PolicyName))
					continue
				}

				// ── Governance allowed: execute tool ──
				logger.Governance(cfg.Agent, tc.Name, tc.Params, decision.Allowed, decision.PolicyName, decision.Reason)
				toolResult := tools.ExecuteDirect(tc.Name, tc.Params, engine.GetTimeout())
				logger.ToolResult(cfg.Agent, tc.Name, toolResult.Success, toolResult.Output)

				var msg string
				if toolResult.Success {
					msg = fmt.Sprintf("Tool %q returned:\n%s", tc.Name, toolResult.Output)
				} else {
					msg = fmt.Sprintf("Tool %q failed: %s", tc.Name, toolResult.Output)
				}
				messages = append(messages, llm.Message{
					Role:       "tool_result",
					Content:    msg,
					ToolCallID: tc.ID,
				})

				log = append(log, fmt.Sprintf("[turn %d] %s → %s", turn, tc.Name, boolStr(toolResult.Success, "ok", "fail")))
			}

			// Last turn: force final answer
			if turn == cfg.MaxTurns {
				messages = append(messages, llm.Message{
					Role:    "user",
					Content: "You've used all turns. Give your final answer now.",
				})
				finalMsgs := compactLLMMessages(messages, cfg.TokenBudget)
				finalResp, ferr := cfg.Provider.Chat(finalMsgs, toolDefs)
				if ferr == nil {
					result.Output = finalResp.Content
					result.PromptTok += finalResp.PromptTok
					result.ResponseTok += finalResp.OutputTok
				}
				result.Turns = turn
				result.ExitReason = "max_turns"
			}
			continue
		}

		// ── No tool calls: final answer (end_turn) ──
		result.Output = provResp.Content
		result.Turns = turn
		result.ExitReason = "final_answer"
		logger.Agent(cfg.Agent, fmt.Sprintf("done — %d turns, %d tool calls", turn, result.ToolCalls))
		break
	}

	result.DurationMs = time.Since(start).Milliseconds()
	result.Success = result.ExitReason == "final_answer"
	result.Log = log
	return result, nil
}

// ----------------------------------------------------------------------------
// Ollama/legacy path: uses []ollama.ChatMessage + intent.Parse()
// ----------------------------------------------------------------------------

func runOllamaLoop(cfg LoopConfig, engine *governance.Engine, start time.Time) (*RunResult, error) {
	runID := fmt.Sprintf("run_%d", time.Now().UnixMilli())
	var seq int
	corrector := correction.NewEngine(3, 10)

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
			result.Turns = turn - 1
			result.ExitReason = "timeout"
			break
		}

		compacted := compactMessages(messages, cfg.TokenBudget)

		tokEst := estimateTokens(compacted)
		logger.Agent(cfg.Agent, fmt.Sprintf("turn %d/%d (~%d tokens)", turn, cfg.MaxTurns, tokEst))

		resp, err := ollama.Chat(compacted, cfg.Model)
		if err != nil {
			logger.Error(cfg.Agent, "ollama: "+err.Error())
			result.Output = "Model error: " + err.Error()
			result.Turns = turn
			result.ExitReason = "model_error"
			break
		}

		content := resp.Message.Content
		promptTok := resp.PromptEval
		outputTok := resp.EvalCount
		totalDurMs := resp.TotalDuration / 1_000_000

		result.PromptTok += promptTok
		result.ResponseTok += outputTok
		logger.ModelCall(cfg.Agent, promptTok, outputTok, totalDurMs)

		messages = append(messages, ollama.ChatMessage{Role: "assistant", Content: content})

		// ── Intent parser: extract action from ANY format ──
		parsed := intent.Parse(content)
		if parsed == nil {
			result.Output = content
			result.Turns = turn
			result.ExitReason = "final_answer"
			logger.Agent(cfg.Agent, fmt.Sprintf("done — %d turns, %d tool calls", turn, result.ToolCalls))
			break
		}

		logger.Agent(cfg.Agent, fmt.Sprintf("intent: %s via %s", parsed.Tool, parsed.Source))

		result.ToolCalls++
		seq++

		proposal := normalizer.Normalize(runID, seq, cfg.Agent, parsed.Tool, parsed.Params)
		fp := normalizer.Fingerprint(proposal)

		canAttempt, skipReason := corrector.ShouldCorrect(fp)
		if !canAttempt {
			logger.Agent(cfg.Agent, fmt.Sprintf("action skipped: %s", skipReason))
			messages = append(messages, ollama.ChatMessage{
				Role:    "user",
				Content: fmt.Sprintf("Tool %q was skipped: %s. Try a different approach.", parsed.Tool, skipReason),
			})
			summary := fmt.Sprintf("[turn %d] %s → skipped (%s)", turn, parsed.Tool, skipReason)
			log = append(log, summary)
			continue
		}

		decision := engine.Evaluate(parsed.Tool, parsed.Params)

		if !decision.Allowed {
			result.Denials++

			govDecision := action.GovernanceDecision{
				Allowed:  false,
				Decision: "deny",
				Reason:   decision.Reason,
				Rule:     decision.PolicyName,
			}

			corrector.RecordDenial(fp, govDecision)
			logger.Governance(cfg.Agent, parsed.Tool, parsed.Params, decision.Allowed, decision.PolicyName, decision.Reason)

			canCorrect, _ := corrector.ShouldCorrect(fp)
			if canCorrect {
				feedback := corrector.BuildFeedback(proposal, govDecision)
				logger.Agent(cfg.Agent, fmt.Sprintf("governance denied %q — sending correction feedback (escalation: %s)", parsed.Tool, corrector.Level()))
				messages = append(messages, ollama.ChatMessage{
					Role:    "user",
					Content: feedback,
				})
			} else {
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
			result.Turns = turn
			result.ExitReason = "max_turns"
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	result.Success = result.ExitReason == "final_answer"
	result.Log = log
	return result, nil
}

// Old parseToolCall/tryParse removed — replaced by intent.Parse()
// which handles all formats: JSON blocks, XML tags, bare JSON,
// OpenAI function_call format, and tool name/param aliasing.

func buildSystemPrompt(base string) string {
	toolDocs := tools.FormatForPrompt()
	return base + `

## Tools

You MUST use tools to complete tasks. Do NOT describe what you would do — actually do it.

To call a tool, output EXACTLY this format (JSON in a code block):

` + "```json\n{\"tool\": \"list_files\", \"params\": {\"directory\": \".\"}}\n```" + `

## Example

User: "What files are in this project?"

WRONG (do not do this):
"I would use list_files to check the directory structure."

CORRECT (do this):
` + "```json\n{\"tool\": \"list_files\", \"params\": {\"directory\": \".\"}}\n```" + `

## Another Example

User: "Read the README"

CORRECT:
` + "```json\n{\"tool\": \"read_file\", \"params\": {\"path\": \"README.md\"}}\n```" + `

## Available tools:

` + toolDocs + `

## Rules
- ALWAYS use tools. Never just describe what you would do.
- Output ONE tool call per response as a JSON code block.
- Wait for the tool result before calling another tool.
- When you have enough information to answer, respond normally WITHOUT a JSON block.
- If governance denies a tool call, try a different approach.
`
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

// compactLLMMessages is the []llm.Message equivalent of compactMessages.
func compactLLMMessages(msgs []llm.Message, budget int) []llm.Message {
	if budget <= 0 {
		budget = 3000
	}
	total := estimateLLMTokens(msgs)
	if total <= budget {
		return msgs
	}

	result := []llm.Message{msgs[0], msgs[1]}
	remaining := msgs[2:]

	for total > budget && len(remaining) > 4 {
		remaining = remaining[2:]
		total = estimateLLMTokens(append(result, remaining...))
	}
	return append(result, remaining...)
}

// estimateLLMTokens estimates token count for []llm.Message.
func estimateLLMTokens(msgs []llm.Message) int {
	total := 0
	for _, m := range msgs {
		total += len(m.Content) / 4
	}
	return total
}

func boolStr(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}

// toProviderMessages converts []ollama.ChatMessage to []llm.Message for use
// with the Provider interface.
func toProviderMessages(msgs []ollama.ChatMessage) []llm.Message {
	result := make([]llm.Message, len(msgs))
	for i, m := range msgs {
		result[i] = llm.Message{Role: m.Role, Content: m.Content}
	}
	return result
}

// buildToolDefs converts tools.Definitions into []llm.ToolDef for the Provider.
func buildToolDefs() []llm.ToolDef {
	defs := make([]llm.ToolDef, len(tools.Definitions))
	for i, d := range tools.Definitions {
		// Build JSON Schema from Param definitions.
		properties := make(map[string]any, len(d.Params))
		required := make([]string, 0, len(d.Params))
		for _, p := range d.Params {
			properties[p.Name] = map[string]any{
				"type":        p.Type,
				"description": p.Desc,
			}
			if p.Required {
				required = append(required, p.Name)
			}
		}

		defs[i] = llm.ToolDef{
			Name:        d.Name,
			Description: d.Description,
			Parameters: map[string]any{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		}
	}
	return defs
}
