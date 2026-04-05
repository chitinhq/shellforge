package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/chitinhq/shellforge/internal/llm"
)

const defaultMaxBytes int64 = 256 * 1024 // 256KB

// Store manages session persistence via JSONL append-log with snapshots.
type Store struct {
	dir      string // directory for session files
	maxBytes int64  // rotation threshold (default 256KB)
}

// New creates a Store that persists sessions to the given directory.
func New(dir string) *Store {
	return &Store{
		dir:      dir,
		maxBytes: defaultMaxBytes,
	}
}

// Entry is one line in the JSONL log.
type Entry struct {
	Type      string            `json:"type"`                 // "message", "meta", "snapshot"
	Timestamp time.Time         `json:"ts"`
	SessionID string            `json:"session_id,omitempty"`
	Message   *llm.Message      `json:"message,omitempty"`   // for type "message"
	Meta      map[string]string `json:"meta,omitempty"`      // for type "meta"
	Messages  []llm.Message     `json:"messages,omitempty"`  // for type "snapshot"
}

// Append writes a message to the session log (append-only, fast).
func (s *Store) Append(sessionID string, msg llm.Message) error {
	entry := Entry{
		Type:      "message",
		Timestamp: time.Now(),
		SessionID: sessionID,
		Message:   &msg,
	}
	if err := s.appendEntry(sessionID, entry); err != nil {
		return err
	}
	return s.maybeRotate(sessionID)
}

// WriteMeta writes session metadata (agent name, model, start time, etc.).
func (s *Store) WriteMeta(sessionID string, meta map[string]string) error {
	entry := Entry{
		Type:      "meta",
		Timestamp: time.Now(),
		SessionID: sessionID,
		Meta:      meta,
	}
	return s.appendEntry(sessionID, entry)
}

// Snapshot writes a full snapshot of the current message history.
// The write is atomic: data is written to a temp file then renamed.
func (s *Store) Snapshot(sessionID string, messages []llm.Message) error {
	entry := Entry{
		Type:      "snapshot",
		Timestamp: time.Now(),
		SessionID: sessionID,
		Messages:  messages,
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("session: marshal snapshot: %w", err)
	}
	line = append(line, '\n')

	// Atomic write: temp file then rename.
	target := s.sessionPath(sessionID)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, line, 0644); err != nil {
		return fmt.Errorf("session: write temp snapshot: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("session: rename snapshot: %w", err)
	}
	return nil
}

// Load reads all entries for a session and reconstructs the message history.
// If a snapshot exists, it loads from the most recent snapshot.
// Returns the messages and metadata.
func (s *Store) Load(sessionID string) ([]llm.Message, map[string]string, error) {
	path := s.sessionPath(sessionID)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("session: open %s: %w", path, err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	// Increase buffer for large snapshots.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, nil, fmt.Errorf("session: decode entry: %w", err)
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("session: scan %s: %w", path, err)
	}

	// Reconstruct: find the most recent snapshot, then replay messages after it.
	var messages []llm.Message
	meta := make(map[string]string)
	snapshotIdx := -1

	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Type == "snapshot" {
			snapshotIdx = i
			break
		}
	}

	startIdx := 0
	if snapshotIdx >= 0 {
		messages = append(messages, entries[snapshotIdx].Messages...)
		startIdx = snapshotIdx + 1
	}

	for i := startIdx; i < len(entries); i++ {
		switch entries[i].Type {
		case "message":
			if entries[i].Message != nil {
				messages = append(messages, *entries[i].Message)
			}
		case "meta":
			for k, v := range entries[i].Meta {
				meta[k] = v
			}
		}
	}

	// Also collect metadata from before the snapshot.
	for i := 0; i < snapshotIdx; i++ {
		if entries[i].Type == "meta" {
			for k, v := range entries[i].Meta {
				meta[k] = v
			}
		}
	}

	return messages, meta, nil
}

// sessionPath returns the JSONL file path for a session.
func (s *Store) sessionPath(sessionID string) string {
	return filepath.Join(s.dir, sessionID+".jsonl")
}

// appendEntry marshals an entry and appends it to the session JSONL file.
func (s *Store) appendEntry(sessionID string, entry Entry) error {
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("session: marshal entry: %w", err)
	}
	line = append(line, '\n')

	path := s.sessionPath(sessionID)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("session: open %s: %w", path, err)
	}
	defer f.Close()

	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("session: write %s: %w", path, err)
	}
	return nil
}

// maybeRotate checks if the session file exceeds maxBytes and rotates if so.
// Rotation means: load current state, write a snapshot atomically, replacing the file.
func (s *Store) maybeRotate(sessionID string) error {
	path := s.sessionPath(sessionID)
	info, err := os.Stat(path)
	if err != nil {
		return nil // file doesn't exist yet, nothing to rotate
	}
	if info.Size() < s.maxBytes {
		return nil
	}

	messages, _, err := s.Load(sessionID)
	if err != nil {
		return fmt.Errorf("session: rotate load: %w", err)
	}
	return s.Snapshot(sessionID, messages)
}
