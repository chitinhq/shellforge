package ralph

import (
	"path/filepath"
	"testing"
)

func TestTaskLog_AppendAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.jsonl")

	log := NewTaskLog(path)

	// Append two entries
	err := log.Append(TaskLogEntry{
		TaskID:      "1",
		Description: "First task",
		Status:      StatusCompleted,
		Output:      "done",
		Turns:       3,
		ToolCalls:   5,
		DurationMs:  1200,
		Timestamp:   "2026-03-31T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("Append 1: %v", err)
	}

	err = log.Append(TaskLogEntry{
		TaskID:      "2",
		Description: "Second task",
		Status:      StatusFailed,
		Error:       "timeout",
		Turns:       10,
		ToolCalls:   8,
		DurationMs:  5000,
		Timestamp:   "2026-03-31T10:01:00Z",
	})
	if err != nil {
		t.Fatalf("Append 2: %v", err)
	}

	// Read back
	entries, err := log.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].TaskID != "1" || entries[0].Status != StatusCompleted {
		t.Errorf("entry 0: %+v", entries[0])
	}
	if entries[1].TaskID != "2" || entries[1].Status != StatusFailed {
		t.Errorf("entry 1: %+v", entries[1])
	}
	if entries[1].Error != "timeout" {
		t.Errorf("entry 1 error: expected 'timeout', got %q", entries[1].Error)
	}
}

func TestTaskLog_Read_NonExistent(t *testing.T) {
	log := NewTaskLog("/nonexistent/ralph.jsonl")
	entries, err := log.Read()
	if err != nil {
		t.Fatalf("Read non-existent: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestTaskLog_Append_SetsTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ralph.jsonl")

	log := NewTaskLog(path)

	// Append without explicit timestamp
	err := log.Append(TaskLogEntry{
		TaskID:      "1",
		Description: "Auto-timestamp",
		Status:      StatusCompleted,
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	entries, _ := log.Read()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Timestamp == "" {
		t.Error("expected auto-generated timestamp")
	}
}
