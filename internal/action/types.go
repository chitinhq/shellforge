// Package action defines the core types for the governed multi-agent
// architecture. Every agent action flows through ActionProposal →
// governance decision → ActionResult. This is the foundational
// contract that all orchestrator phases build on.
package action

// RiskLevel classifies the danger of an action for governance evaluation.
type RiskLevel string

const (
	RiskReadOnly    RiskLevel = "read_only"
	RiskMutating    RiskLevel = "mutating"
	RiskDestructive RiskLevel = "destructive"
)

// Scope defines the blast radius of an action.
type Scope string

const (
	ScopeFile       Scope = "file"
	ScopeDirectory  Scope = "directory"
	ScopeRepository Scope = "repository"
	ScopeSystem     Scope = "system"
)

// ActionType enumerates the specific operations agents can propose.
type ActionType string

const (
	FileRead    ActionType = "file.read"
	FileWrite   ActionType = "file.write"
	FileDelete  ActionType = "file.delete"
	ShellExec   ActionType = "shell.exec"
	GitDiff     ActionType = "git.diff"
	GitCommit   ActionType = "git.commit"
	GitPush     ActionType = "git.push"
	HTTPRequest ActionType = "http.request"
)

// Proposal represents an agent's intent to perform an action.
// It must be evaluated by the governance engine before execution.
type Proposal struct {
	ID       string         `json:"id"`
	RunID    string         `json:"run_id"`
	Sequence int            `json:"sequence"`
	Agent    string         `json:"agent"`
	Type     ActionType     `json:"type"`
	Target   string         `json:"target"`
	Params   map[string]any `json:"params"`
	Risk     RiskLevel      `json:"risk"`
	Scope    Scope          `json:"scope"`
	Timeout  int            `json:"timeout_ms"`
}

// Status represents the outcome of an action execution.
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailure Status = "failure"
	StatusDenied  Status = "denied"
	StatusTimeout Status = "timeout"
	StatusBlocked Status = "blocked"
	StatusSkipped Status = "skipped"
)

// GovernanceDecision records why an action was allowed or denied.
type GovernanceDecision struct {
	Allowed     bool           `json:"allowed"`
	Decision    string         `json:"decision"`
	Reason      string         `json:"reason"`
	Rule        string         `json:"rule"`
	Severity    int            `json:"severity"`
	Suggestion  string         `json:"suggestion"`
	Constraints map[string]any `json:"constraints,omitempty"`
}

// Result captures the outcome of executing a governed action.
type Result struct {
	ProposalID string             `json:"proposal_id"`
	Status     Status             `json:"status"`
	Output     string             `json:"output"`
	Error      string             `json:"error,omitempty"`
	DurationMs int                `json:"duration_ms"`
	Governance GovernanceDecision `json:"governance"`
}
