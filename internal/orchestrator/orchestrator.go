package orchestrator

import (
	"fmt"
	"time"

	"github.com/chitinhq/shellforge/internal/agent"
	"github.com/chitinhq/shellforge/internal/governance"
	"github.com/chitinhq/shellforge/internal/llm"
)

// Orchestrator manages sub-agent execution with concurrency control.
// Each sub-agent gets its own RunLoop call with isolated context.
type Orchestrator struct {
	provider    llm.Provider
	governance  *governance.Engine
	maxParallel int
	slots       chan struct{}
}

// NewOrchestrator creates an Orchestrator with the given concurrency limit.
func NewOrchestrator(provider llm.Provider, gov *governance.Engine, maxParallel int) *Orchestrator {
	if maxParallel < 1 {
		maxParallel = 1
	}
	slots := make(chan struct{}, maxParallel)
	for i := 0; i < maxParallel; i++ {
		slots <- struct{}{}
	}
	return &Orchestrator{
		provider:    provider,
		governance:  gov,
		maxParallel: maxParallel,
		slots:       slots,
	}
}

// SpawnSync executes a sub-agent synchronously, blocking until completion.
// Acquires a concurrency slot before running.
func (o *Orchestrator) SpawnSync(task SubTask) (*SubResult, error) {
	// Acquire slot
	<-o.slots
	defer func() { o.slots <- struct{}{} }()

	return o.executeTask(task)
}

// SpawnAsync launches a sub-agent in a goroutine and returns a handle.
// The handle can be passed to Collect() to retrieve the result.
func (o *Orchestrator) SpawnAsync(task SubTask) (TaskHandle, error) {
	handle := TaskHandle{
		TaskID: task.ID,
		done:   make(chan *asyncResult, 1),
	}

	go func() {
		// Acquire slot
		<-o.slots
		defer func() { o.slots <- struct{}{} }()

		result, err := o.executeTask(task)
		handle.done <- &asyncResult{result: result, err: err}
	}()

	return handle, nil
}

// Collect blocks until the async task completes or the timeout expires.
func (o *Orchestrator) Collect(h TaskHandle, timeout time.Duration) (*SubResult, error) {
	select {
	case ar := <-h.done:
		if ar.err != nil {
			return nil, ar.err
		}
		return ar.result, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("collect timeout for task %s after %s", h.TaskID, timeout)
	}
}

// executeTask runs a single sub-agent via agent.RunLoop.
func (o *Orchestrator) executeTask(task SubTask) (*SubResult, error) {
	cfg := agent.LoopConfig{
		Agent:       fmt.Sprintf("sub-agent-%s", task.ID),
		System:      task.System,
		UserPrompt:  task.Description,
		Model:       task.Model,
		MaxTurns:    task.MaxTurns,
		TimeoutMs:   task.TimeoutMs,
		OutputDir:   "",
		TokenBudget: task.TokenBudget,
		Provider:    o.provider,
	}

	if cfg.System == "" {
		cfg.System = "You are a sub-agent. Complete the requested task precisely."
	}
	if cfg.MaxTurns == 0 {
		cfg.MaxTurns = 10
	}
	if cfg.TimeoutMs == 0 {
		cfg.TimeoutMs = 60000
	}
	if cfg.TokenBudget == 0 {
		cfg.TokenBudget = 3000
	}

	start := time.Now()
	runResult, err := agent.RunLoop(cfg, o.governance)
	if err != nil {
		return &SubResult{
			TaskID:     task.ID,
			Success:    false,
			Error:      err.Error(),
			DurationMs: time.Since(start).Milliseconds(),
		}, err
	}

	output := CompressResult(runResult.Output)

	return &SubResult{
		TaskID:     task.ID,
		Success:    runResult.Success,
		Output:     output,
		Turns:      runResult.Turns,
		ToolCalls:  runResult.ToolCalls,
		DurationMs: runResult.DurationMs,
	}, nil
}
