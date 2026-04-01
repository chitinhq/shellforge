package orchestrator

// SubTask describes a unit of work to be executed by a sub-agent.
// Each sub-agent gets its own RunLoop call with isolated context.
type SubTask struct {
	ID          string
	Description string
	System      string // system prompt for the sub-agent
	Model       string
	MaxTurns    int
	TimeoutMs   int
	TokenBudget int
}

// SubResult captures the outcome of a sub-agent execution.
type SubResult struct {
	TaskID     string
	Success    bool
	Output     string
	Turns      int
	ToolCalls  int
	DurationMs int64
	Error      string
}

// TaskHandle is a reference to an in-flight async sub-agent task.
// Use Collect() to block until the result is available.
type TaskHandle struct {
	TaskID string
	done   chan *asyncResult
}

// asyncResult wraps a SubResult with an optional error from the agent call.
type asyncResult struct {
	result *SubResult
	err    error
}
