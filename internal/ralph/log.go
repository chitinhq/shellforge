package ralph

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// TaskLogEntry records the result of a single task execution.
type TaskLogEntry struct {
	TaskID      string     `json:"task_id"`
	Description string     `json:"description"`
	Status      TaskStatus `json:"status"`
	Output      string     `json:"output,omitempty"`
	Error       string     `json:"error,omitempty"`
	Turns       int        `json:"turns"`
	ToolCalls   int        `json:"tool_calls"`
	DurationMs  int64      `json:"duration_ms"`
	Timestamp   string     `json:"timestamp"`
}

// TaskLog is an append-only JSONL log of task results.
type TaskLog struct {
	Path string
}

// NewTaskLog creates a TaskLog that writes to the given file path.
func NewTaskLog(path string) *TaskLog {
	return &TaskLog{Path: path}
}

// Append writes a single log entry as a JSONL line.
func (tl *TaskLog) Append(entry TaskLogEntry) error {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}
	f, err := os.OpenFile(tl.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close()
	_, err = f.Write(append(data, '\n'))
	if err != nil {
		return fmt.Errorf("write log entry: %w", err)
	}
	return nil
}

// Read returns all log entries from the file.
func (tl *TaskLog) Read() ([]TaskLogEntry, error) {
	data, err := os.ReadFile(tl.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read log file: %w", err)
	}

	var entries []TaskLogEntry
	// Split by newlines and parse each line
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			line := data[start:i]
			start = i + 1
			if len(line) == 0 {
				continue
			}
			var entry TaskLogEntry
			if err := json.Unmarshal(line, &entry); err != nil {
				continue // skip malformed lines
			}
			entries = append(entries, entry)
		}
	}
	// Handle last line without trailing newline
	if start < len(data) {
		line := data[start:]
		var entry TaskLogEntry
		if err := json.Unmarshal(line, &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	return entries, nil
}
