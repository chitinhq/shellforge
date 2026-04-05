package ralph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chitinhq/shellforge/internal/agent"
)

func TestRalphConfig_Defaults(t *testing.T) {
	cfg := RalphConfig{
		TaskSource: SourceFile,
		TaskFile:   "tasks.json",
	}
	if cfg.TaskSource != SourceFile {
		t.Errorf("expected file source, got %s", cfg.TaskSource)
	}
}

func TestMakePicker_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tasks.json")
	os.WriteFile(path, []byte("[]"), 0o644)

	cfg := RalphConfig{TaskSource: SourceFile, TaskFile: path}
	picker, err := makePicker(cfg)
	if err != nil {
		t.Fatalf("makePicker: %v", err)
	}
	if picker == nil {
		t.Fatal("expected non-nil picker")
	}
}

func TestMakePicker_NoFile(t *testing.T) {
	cfg := RalphConfig{TaskSource: SourceFile, TaskFile: ""}
	_, err := makePicker(cfg)
	if err == nil {
		t.Fatal("expected error for empty task file")
	}
}

func TestMakePicker_MCP(t *testing.T) {
	cfg := RalphConfig{TaskSource: SourceMCP}
	_, err := makePicker(cfg)
	if err == nil {
		t.Fatal("expected error for unimplemented MCP source")
	}
}

func TestMakePicker_Unknown(t *testing.T) {
	cfg := RalphConfig{TaskSource: "unknown"}
	_, err := makePicker(cfg)
	if err == nil {
		t.Fatal("expected error for unknown source type")
	}
}

func TestRunRalph_DryRun(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "tasks.json")
	logPath := filepath.Join(dir, "ralph.jsonl")

	tasks := `[
  {"id": "1", "description": "Task one", "status": "pending", "priority": 1},
  {"id": "2", "description": "Task two", "status": "pending", "priority": 2}
]`
	os.WriteFile(taskPath, []byte(tasks), 0o644)

	cfg := RalphConfig{
		TaskSource: SourceFile,
		TaskFile:   taskPath,
		LogFile:    logPath,
		DryRun:     true,
		MaxTasks:   10,
		LoopConfig: agent.LoopConfig{
			Agent:       "test-agent",
			System:      "test",
			MaxTurns:    5,
			TimeoutMs:   10000,
			OutputDir:   dir,
			TokenBudget: 1000,
		},
	}

	// In dry-run, we don't need a real governance engine
	result, err := RunRalph(cfg, nil)
	if err != nil {
		t.Fatalf("RunRalph dry-run: %v", err)
	}
	if result.Skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", result.Skipped)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Total)
	}

	// Verify tasks are marked completed (dry-run advances past them)
	readTasks, _ := ParseTaskFile(taskPath)
	for _, task := range readTasks {
		if task.Status != StatusCompleted {
			t.Errorf("task %s should be completed after dry-run, got %s", task.ID, task.Status)
		}
	}
}

func TestRunRalph_MaxTasks(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "tasks.json")
	logPath := filepath.Join(dir, "ralph.jsonl")

	tasks := `[
  {"id": "1", "description": "Task one", "status": "pending", "priority": 1},
  {"id": "2", "description": "Task two", "status": "pending", "priority": 2},
  {"id": "3", "description": "Task three", "status": "pending", "priority": 3}
]`
	os.WriteFile(taskPath, []byte(tasks), 0o644)

	cfg := RalphConfig{
		TaskSource: SourceFile,
		TaskFile:   taskPath,
		LogFile:    logPath,
		DryRun:     true,
		MaxTasks:   2, // only process 2
		LoopConfig: agent.LoopConfig{
			Agent:       "test-agent",
			System:      "test",
			MaxTurns:    5,
			TimeoutMs:   10000,
			OutputDir:   dir,
			TokenBudget: 1000,
		},
	}

	result, err := RunRalph(cfg, nil)
	if err != nil {
		t.Fatalf("RunRalph max-tasks: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 total (max limit), got %d", result.Total)
	}
}

func TestRunRalph_NoPendingTasks(t *testing.T) {
	dir := t.TempDir()
	taskPath := filepath.Join(dir, "tasks.json")
	logPath := filepath.Join(dir, "ralph.jsonl")

	tasks := `[{"id": "1", "description": "Done", "status": "completed", "priority": 1}]`
	os.WriteFile(taskPath, []byte(tasks), 0o644)

	cfg := RalphConfig{
		TaskSource: SourceFile,
		TaskFile:   taskPath,
		LogFile:    logPath,
		LoopConfig: agent.LoopConfig{
			Agent:       "test-agent",
			System:      "test",
			MaxTurns:    5,
			TimeoutMs:   10000,
			OutputDir:   dir,
			TokenBudget: 1000,
		},
	}

	result, err := RunRalph(cfg, nil)
	if err != nil {
		t.Fatalf("RunRalph no-pending: %v", err)
	}
	if result.Total != 0 {
		t.Errorf("expected 0 total, got %d", result.Total)
	}
}
