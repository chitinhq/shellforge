// Package ralph implements the Ralph Loop — a stateless-iterative
// execution pattern for autonomous agent task processing. Each cycle:
// PICK → IMPLEMENT → VALIDATE → COMMIT → RESET.
//
// The loop is stateless across iterations: each task gets a fresh
// RunLoop call with no prior message history, preventing context
// pollution between tasks.
package ralph

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// TaskStatus represents the lifecycle state of a task.
type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
)

// Task is a single unit of work for the Ralph Loop.
type Task struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Priority    int        `json:"priority"`
	Error       string     `json:"error,omitempty"`
}

// ParseTaskFile reads a JSON task file and returns the task list.
func ParseTaskFile(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read task file: %w", err)
	}
	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse task file: %w", err)
	}
	return tasks, nil
}

// WriteTaskFile writes the task list back to disk as JSON.
func WriteTaskFile(path string, tasks []Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write task file: %w", err)
	}
	return nil
}

// NextPending returns the highest-priority pending task (lowest priority number).
// Returns nil if no pending tasks remain.
func NextPending(tasks []Task) *Task {
	var pending []Task
	for _, t := range tasks {
		if t.Status == StatusPending {
			pending = append(pending, t)
		}
	}
	if len(pending) == 0 {
		return nil
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Priority < pending[j].Priority
	})
	result := pending[0]
	return &result
}
