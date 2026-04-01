package ralph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTaskFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	content := `[
  {"id": "1", "description": "Add input validation", "status": "pending", "priority": 1},
  {"id": "2", "description": "Write tests", "status": "pending", "priority": 2},
  {"id": "3", "description": "Fix bug", "status": "completed", "priority": 0}
]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	tasks, err := ParseTaskFile(path)
	if err != nil {
		t.Fatalf("ParseTaskFile: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(tasks))
	}

	if tasks[0].ID != "1" || tasks[0].Status != StatusPending {
		t.Errorf("task 0: got id=%s status=%s", tasks[0].ID, tasks[0].Status)
	}
	if tasks[2].Status != StatusCompleted {
		t.Errorf("task 2: expected completed, got %s", tasks[2].Status)
	}
}

func TestParseTaskFile_NotFound(t *testing.T) {
	_, err := ParseTaskFile("/nonexistent/tasks.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseTaskFile_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	os.WriteFile(path, []byte("not json"), 0o644)

	_, err := ParseTaskFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestWriteTaskFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	tasks := []Task{
		{ID: "1", Description: "Test task", Status: StatusPending, Priority: 1},
	}
	if err := WriteTaskFile(path, tasks); err != nil {
		t.Fatalf("WriteTaskFile: %v", err)
	}

	// Read back
	readTasks, err := ParseTaskFile(path)
	if err != nil {
		t.Fatalf("ParseTaskFile after write: %v", err)
	}
	if len(readTasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(readTasks))
	}
	if readTasks[0].ID != "1" || readTasks[0].Description != "Test task" {
		t.Errorf("round-trip mismatch: %+v", readTasks[0])
	}
}

func TestNextPending(t *testing.T) {
	tasks := []Task{
		{ID: "1", Description: "Low priority", Status: StatusPending, Priority: 3},
		{ID: "2", Description: "High priority", Status: StatusPending, Priority: 1},
		{ID: "3", Description: "Already done", Status: StatusCompleted, Priority: 0},
		{ID: "4", Description: "Medium priority", Status: StatusPending, Priority: 2},
	}

	next := NextPending(tasks)
	if next == nil {
		t.Fatal("expected a pending task")
	}
	if next.ID != "2" {
		t.Errorf("expected task 2 (highest priority), got task %s", next.ID)
	}
}

func TestNextPending_NoPending(t *testing.T) {
	tasks := []Task{
		{ID: "1", Status: StatusCompleted, Priority: 1},
		{ID: "2", Status: StatusFailed, Priority: 2},
	}

	next := NextPending(tasks)
	if next != nil {
		t.Errorf("expected nil, got task %s", next.ID)
	}
}

func TestNextPending_EmptyList(t *testing.T) {
	next := NextPending(nil)
	if next != nil {
		t.Error("expected nil for empty list")
	}
}
