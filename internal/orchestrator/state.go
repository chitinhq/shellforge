// Package orchestrator implements the state machine that governs
// multi-agent run lifecycle. Every run transitions through explicit
// phases — there are no implicit state changes. Invalid transitions
// are rejected, making the system auditable and debuggable.
package orchestrator

import (
	"fmt"
	"time"

	"github.com/chitinhq/shellforge/internal/action"
)

// Phase represents a discrete stage in the orchestrator lifecycle.
type Phase string

const (
	PhaseIdle       Phase = "IDLE"
	PhasePlanning   Phase = "PLANNING"
	PhaseWorking    Phase = "WORKING"
	PhaseEvaluating Phase = "EVALUATING"
	PhaseCorrecting Phase = "CORRECTING"
	PhaseComplete   Phase = "COMPLETE"
	PhaseFailed     Phase = "FAILED"
)

// RunState tracks everything about an in-progress orchestration run.
type RunState struct {
	RunID        string
	Phase        Phase
	Task         string
	Plan         []action.Proposal
	Results      []action.Result
	Denials      []action.GovernanceDecision
	TotalDenials int
	RetryCount   int
	MaxRetries   int
	StartTime    time.Time
}

// validTransitions defines the allowed state machine edges.
// Terminal states (Complete, Failed) have no outgoing transitions.
var validTransitions = map[Phase][]Phase{
	PhaseIdle:       {PhasePlanning},
	PhasePlanning:   {PhaseWorking, PhaseFailed},
	PhaseWorking:    {PhaseEvaluating, PhaseFailed},
	PhaseEvaluating: {PhaseComplete, PhaseCorrecting, PhaseFailed},
	PhaseCorrecting: {PhaseWorking, PhaseFailed},
}

// NewRunState creates a fresh run in the IDLE phase.
func NewRunState(runID, task string, maxRetries int) *RunState {
	return &RunState{
		RunID:      runID,
		Phase:      PhaseIdle,
		Task:       task,
		MaxRetries: maxRetries,
		StartTime:  time.Now(),
	}
}

// NewRun creates a RunState with an auto-generated ID and default settings.
// This is a convenience constructor for callers that do not need to control
// the run ID or retry budget.
func NewRun(task string) *RunState {
	runID := fmt.Sprintf("run_%d", time.Now().UnixMilli())
	return NewRunState(runID, task, 3)
}

// Transition moves the run to a new phase if the transition is valid.
// Returns an error for invalid transitions, preventing illegal state changes.
func (s *RunState) Transition(to Phase) error {
	allowed := validTransitions[s.Phase]
	for _, p := range allowed {
		if p == to {
			s.Phase = to
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s → %s", s.Phase, to)
}

// AddResult appends an action result and tracks denial statistics.
func (s *RunState) AddResult(r action.Result) {
	s.Results = append(s.Results, r)
	if !r.Governance.Allowed {
		s.Denials = append(s.Denials, r.Governance)
		s.TotalDenials++
	}
}

// IsTerminal returns true if the run is in a final state.
func (s *RunState) IsTerminal() bool {
	return s.Phase == PhaseComplete || s.Phase == PhaseFailed
}

// Elapsed returns the wall-clock duration since the run started.
func (s *RunState) Elapsed() time.Duration {
	return time.Since(s.StartTime)
}
