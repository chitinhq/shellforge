// Package correction handles governance denials by tracking repeated
// offenses, escalating enforcement, and generating corrective feedback
// that guides the LLM toward compliant alternatives.
package correction

import (
	"fmt"

	"github.com/chitinhq/shellforge/internal/action"
)

// EscalationLevel tracks how aggressively the correction engine restricts
// the agent based on cumulative denial count.
type EscalationLevel int

const (
	Normal   EscalationLevel = iota // < 3 denials
	Elevated                        // 3-6
	High                            // 7-9
	Lockdown                        // >= 10
)

// String returns a human-readable label for the escalation level.
func (e EscalationLevel) String() string {
	switch e {
	case Normal:
		return "normal"
	case Elevated:
		return "elevated"
	case High:
		return "high"
	case Lockdown:
		return "lockdown"
	default:
		return "unknown"
	}
}

// Engine handles governance denials by generating corrective feedback.
type Engine struct {
	maxRetries   int
	totalDenials int
	maxTotal     int
	seen         map[string]int // fingerprint -> denial count
	escalation   EscalationLevel
}

// NewEngine creates a correction engine with the given per-action retry
// limit and total denial budget.
func NewEngine(maxRetries, maxTotal int) *Engine {
	return &Engine{
		maxRetries: maxRetries,
		maxTotal:   maxTotal,
		seen:       make(map[string]int),
		escalation: Normal,
	}
}

// ShouldCorrect checks if correction is possible or if we should skip/abort.
// Returns (true, "") if the agent may retry, or (false, reason) if it must stop.
func (e *Engine) ShouldCorrect(fingerprint string) (bool, string) {
	if e.escalation >= Lockdown {
		return false, "agent is in lockdown — too many governance denials"
	}

	count := e.seen[fingerprint]
	if count >= e.maxRetries {
		return false, fmt.Sprintf("action retried %d times (max %d) — skipping", count, e.maxRetries)
	}

	if e.totalDenials >= e.maxTotal {
		return false, fmt.Sprintf("total denial budget exhausted (%d/%d)", e.totalDenials, e.maxTotal)
	}

	return true, ""
}

// RecordDenial tracks a denial and updates escalation state.
func (e *Engine) RecordDenial(fingerprint string, denial action.GovernanceDecision) {
	e.seen[fingerprint]++
	e.totalDenials++
	e.updateEscalation()
}

// BuildFeedback creates a structured prompt for the LLM to correct its action.
func (e *Engine) BuildFeedback(proposal action.Proposal, denial action.GovernanceDecision) string {
	feedback := fmt.Sprintf(
		"Your action was denied by governance policy.\n"+
			"Action: %s → %s\n"+
			"Reason: %s\n",
		string(proposal.Type), proposal.Target, denial.Reason)

	if denial.Suggestion != "" {
		feedback += fmt.Sprintf("Suggestion: %s\n", denial.Suggestion)
	}

	if denial.Rule != "" {
		feedback += fmt.Sprintf("Policy: %s\n", denial.Rule)
	}

	feedback += "\nProduce a different action that achieves the same goal while complying with governance. Do NOT repeat the denied action."

	if e.escalation >= Elevated {
		feedback += fmt.Sprintf("\n\nWARNING: Escalation level is %s (%d total denials). Further violations may cause the agent to be locked out.", e.escalation, e.totalDenials)
	}

	return feedback
}

// Level returns the current escalation level.
func (e *Engine) Level() EscalationLevel {
	return e.escalation
}

// TotalDenials returns the cumulative denial count.
func (e *Engine) TotalDenials() int {
	return e.totalDenials
}

// updateEscalation recalculates the escalation level from totalDenials.
func (e *Engine) updateEscalation() {
	switch {
	case e.totalDenials >= 10:
		e.escalation = Lockdown
	case e.totalDenials >= 7:
		e.escalation = High
	case e.totalDenials >= 3:
		e.escalation = Elevated
	default:
		e.escalation = Normal
	}
}
