package orchestrator

import (
	"testing"
)

func TestSubTask_Fields(t *testing.T) {
	task := SubTask{
		ID:          "sub-1",
		Description: "Analyze code quality",
		System:      "You are a QA agent.",
		Model:       "test-model",
		MaxTurns:    5,
		TimeoutMs:   30000,
		TokenBudget: 2000,
	}

	if task.ID != "sub-1" {
		t.Errorf("expected id sub-1, got %s", task.ID)
	}
	if task.MaxTurns != 5 {
		t.Errorf("expected 5 max turns, got %d", task.MaxTurns)
	}
}

func TestSubResult_Fields(t *testing.T) {
	result := SubResult{
		TaskID:     "sub-1",
		Success:    true,
		Output:     "All tests pass",
		Turns:      3,
		ToolCalls:  5,
		DurationMs: 1500,
	}

	if !result.Success {
		t.Error("expected success")
	}
	if result.TaskID != "sub-1" {
		t.Errorf("expected task id sub-1, got %s", result.TaskID)
	}
}

func TestTaskHandle_Fields(t *testing.T) {
	handle := TaskHandle{
		TaskID: "sub-1",
		done:   make(chan *asyncResult, 1),
	}

	if handle.TaskID != "sub-1" {
		t.Errorf("expected task id sub-1, got %s", handle.TaskID)
	}
}
