package ralph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFilePicker_Pick(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	content := `[
  {"id": "1", "description": "First task", "status": "pending", "priority": 2},
  {"id": "2", "description": "Second task", "status": "pending", "priority": 1},
  {"id": "3", "description": "Done task", "status": "completed", "priority": 0}
]`
	os.WriteFile(path, []byte(content), 0o644)

	picker := NewFilePicker(path)
	task, err := picker.Pick()
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if task == nil {
		t.Fatal("expected a task")
	}
	if task.ID != "2" {
		t.Errorf("expected task 2, got %s", task.ID)
	}
}

func TestFilePicker_Pick_NoPending(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	content := `[{"id": "1", "description": "Done", "status": "completed", "priority": 1}]`
	os.WriteFile(path, []byte(content), 0o644)

	picker := NewFilePicker(path)
	task, err := picker.Pick()
	if err != nil {
		t.Fatalf("Pick: %v", err)
	}
	if task != nil {
		t.Errorf("expected nil, got task %s", task.ID)
	}
}

func TestFilePicker_Update(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")

	content := `[
  {"id": "1", "description": "Task one", "status": "pending", "priority": 1},
  {"id": "2", "description": "Task two", "status": "pending", "priority": 2}
]`
	os.WriteFile(path, []byte(content), 0o644)

	picker := NewFilePicker(path)

	// Update task 1 to completed
	err := picker.Update(Task{
		ID:          "1",
		Description: "Task one",
		Status:      StatusCompleted,
		Priority:    1,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Verify the update persisted
	tasks, err := ParseTaskFile(path)
	if err != nil {
		t.Fatalf("ParseTaskFile after update: %v", err)
	}
	if tasks[0].Status != StatusCompleted {
		t.Errorf("expected completed, got %s", tasks[0].Status)
	}
	if tasks[1].Status != StatusPending {
		t.Errorf("task 2 should still be pending, got %s", tasks[1].Status)
	}
}
