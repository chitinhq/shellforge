package ralph

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/chitinhq/shellforge/internal/agent"
	"github.com/chitinhq/shellforge/internal/governance"
)

// TaskSourceType defines how tasks are sourced.
type TaskSourceType string

const (
	SourceFile TaskSourceType = "file"
	SourceMCP  TaskSourceType = "mcp"
)

// RalphConfig configures a Ralph Loop execution.
type RalphConfig struct {
	TaskSource  TaskSourceType
	TaskFile    string
	MCPEndpoint string
	LogFile     string
	Validate    []string // shell commands to run for validation
	AutoCommit  bool
	MaxTasks    int // 0 = unlimited
	LoopConfig  agent.LoopConfig
	DryRun      bool
}

// RalphResult summarizes the outcome of a Ralph Loop run.
type RalphResult struct {
	Completed int
	Failed    int
	Skipped   int
	Total     int
	Entries   []TaskLogEntry
}

// RunRalph executes the Ralph Loop: PICK -> IMPLEMENT -> VALIDATE -> COMMIT -> RESET.
// Each iteration picks the next pending task, runs agent.RunLoop with a fresh context,
// validates the result, optionally commits, then resets for the next task.
func RunRalph(cfg RalphConfig, engine *governance.Engine) (*RalphResult, error) {
	picker, err := makePicker(cfg)
	if err != nil {
		return nil, fmt.Errorf("create picker: %w", err)
	}

	logFile := cfg.LogFile
	if logFile == "" {
		logFile = "ralph-log.jsonl"
	}
	taskLog := NewTaskLog(logFile)

	result := &RalphResult{}
	processed := 0

	for {
		// Check task limit
		if cfg.MaxTasks > 0 && processed >= cfg.MaxTasks {
			break
		}

		// ── PICK ──
		task, err := picker.Pick()
		if err != nil {
			return result, fmt.Errorf("pick task: %w", err)
		}
		if task == nil {
			break // no more pending tasks
		}
		result.Total++

		// Mark as running
		task.Status = StatusRunning
		picker.Update(*task)

		if cfg.DryRun {
			fmt.Printf("[ralph] DRY RUN — would implement task %s: %s\n", task.ID, task.Description)
			task.Status = StatusCompleted // mark completed so we don't pick it again
			picker.Update(*task)
			result.Skipped++
			processed++
			continue
		}

		// ── IMPLEMENT ──
		loopCfg := cfg.LoopConfig
		loopCfg.UserPrompt = task.Description
		loopCfg.Agent = fmt.Sprintf("ralph-task-%s", task.ID)

		start := time.Now()
		runResult, runErr := agent.RunLoop(loopCfg, engine)

		entry := TaskLogEntry{
			TaskID:      task.ID,
			Description: task.Description,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}

		if runErr != nil {
			task.Status = StatusFailed
			task.Error = runErr.Error()
			entry.Status = StatusFailed
			entry.Error = runErr.Error()
			entry.DurationMs = time.Since(start).Milliseconds()
			picker.Update(*task)
			taskLog.Append(entry)
			result.Failed++
			result.Entries = append(result.Entries, entry)
			processed++
			continue
		}

		entry.Output = runResult.Output
		entry.Turns = runResult.Turns
		entry.ToolCalls = runResult.ToolCalls
		entry.DurationMs = runResult.DurationMs

		// ── VALIDATE ──
		validated := true
		if len(cfg.Validate) > 0 && runResult.Success {
			for _, cmdStr := range cfg.Validate {
				parts := strings.Fields(cmdStr)
				if len(parts) == 0 {
					continue
				}
				cmd := exec.Command(parts[0], parts[1:]...)
				out, verr := cmd.CombinedOutput()
				if verr != nil {
					validated = false
					task.Error = fmt.Sprintf("validation failed (%s): %s", cmdStr, string(out))
					break
				}
			}
		}

		if !runResult.Success || !validated {
			task.Status = StatusFailed
			if task.Error == "" {
				task.Error = fmt.Sprintf("agent exit: %s", runResult.ExitReason)
			}
			entry.Status = StatusFailed
			entry.Error = task.Error
			picker.Update(*task)
			taskLog.Append(entry)
			result.Failed++
			result.Entries = append(result.Entries, entry)
			processed++
			continue
		}

		// ── COMMIT ──
		if cfg.AutoCommit {
			commitMsg := fmt.Sprintf("ralph: task %s — %s", task.ID, task.Description)
			addCmd := exec.Command("git", "add", "-A")
			addCmd.Run()
			commitCmd := exec.Command("git", "commit", "-m", commitMsg, "--allow-empty")
			commitCmd.Run()
		}

		// ── RESET ── (implicit: next iteration creates a fresh RunLoop)
		task.Status = StatusCompleted
		entry.Status = StatusCompleted
		picker.Update(*task)
		taskLog.Append(entry)
		result.Completed++
		result.Entries = append(result.Entries, entry)
		processed++
	}

	return result, nil
}

func makePicker(cfg RalphConfig) (Picker, error) {
	switch cfg.TaskSource {
	case SourceFile, "":
		if cfg.TaskFile == "" {
			return nil, fmt.Errorf("task file path required for file source")
		}
		return NewFilePicker(cfg.TaskFile), nil
	case SourceMCP:
		return nil, fmt.Errorf("MCP task source not yet implemented")
	default:
		return nil, fmt.Errorf("unknown task source: %s", cfg.TaskSource)
	}
}
