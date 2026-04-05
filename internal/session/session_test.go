package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chitinhq/shellforge/internal/llm"
)

func TestAppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	msgs := []llm.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there"},
		{Role: "user", Content: "Run a tool"},
		{Role: "assistant", Content: "", ToolCalls: []llm.ToolCall{
			{ID: "call_1", Name: "read_file", Params: map[string]string{"path": "/tmp/x"}},
		}},
		{Role: "tool_result", Content: "file contents", ToolCallID: "call_1"},
	}

	for _, m := range msgs {
		if err := store.Append("sess-1", m); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, meta, err := store.Load("sess-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(meta) != 0 {
		t.Errorf("expected empty meta, got %v", meta)
	}
	if len(got) != len(msgs) {
		t.Fatalf("message count: got %d, want %d", len(got), len(msgs))
	}
	for i, m := range got {
		if m.Role != msgs[i].Role {
			t.Errorf("msg[%d].Role: got %q, want %q", i, m.Role, msgs[i].Role)
		}
		if m.Content != msgs[i].Content {
			t.Errorf("msg[%d].Content: got %q, want %q", i, m.Content, msgs[i].Content)
		}
		if m.ToolCallID != msgs[i].ToolCallID {
			t.Errorf("msg[%d].ToolCallID: got %q, want %q", i, m.ToolCallID, msgs[i].ToolCallID)
		}
		if len(m.ToolCalls) != len(msgs[i].ToolCalls) {
			t.Errorf("msg[%d].ToolCalls count: got %d, want %d", i, len(m.ToolCalls), len(msgs[i].ToolCalls))
		}
	}
}

func TestWriteMetaAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	meta := map[string]string{
		"agent": "shellforge-sr",
		"model": "claude-opus-4-20250514",
		"start": "2026-04-02T10:00:00Z",
	}
	if err := store.WriteMeta("sess-meta", meta); err != nil {
		t.Fatalf("WriteMeta: %v", err)
	}

	// Also append a message so we can verify both come back.
	if err := store.Append("sess-meta", llm.Message{Role: "user", Content: "hi"}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	msgs, gotMeta, err := store.Load("sess-meta")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("message count: got %d, want 1", len(msgs))
	}
	if msgs[0].Content != "hi" {
		t.Errorf("message content: got %q, want %q", msgs[0].Content, "hi")
	}
	for k, want := range meta {
		if gotMeta[k] != want {
			t.Errorf("meta[%q]: got %q, want %q", k, gotMeta[k], want)
		}
	}
}

func TestSnapshotAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	snapshot := []llm.Message{
		{Role: "user", Content: "first"},
		{Role: "assistant", Content: "second"},
		{Role: "user", Content: "third"},
	}
	if err := store.Snapshot("sess-snap", snapshot); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	got, _, err := store.Load("sess-snap")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != len(snapshot) {
		t.Fatalf("message count: got %d, want %d", len(got), len(snapshot))
	}
	for i, m := range got {
		if m.Role != snapshot[i].Role {
			t.Errorf("msg[%d].Role: got %q, want %q", i, m.Role, snapshot[i].Role)
		}
		if m.Content != snapshot[i].Content {
			t.Errorf("msg[%d].Content: got %q, want %q", i, m.Content, snapshot[i].Content)
		}
	}
}

func TestSnapshotThenAppendAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	// Write a snapshot as the base.
	snapshot := []llm.Message{
		{Role: "user", Content: "original question"},
		{Role: "assistant", Content: "original answer"},
	}
	if err := store.Snapshot("sess-both", snapshot); err != nil {
		t.Fatalf("Snapshot: %v", err)
	}

	// Append additional messages after the snapshot.
	additional := []llm.Message{
		{Role: "user", Content: "follow-up"},
		{Role: "assistant", Content: "follow-up answer"},
	}
	for _, m := range additional {
		if err := store.Append("sess-both", m); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, _, err := store.Load("sess-both")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := append(snapshot, additional...)
	if len(got) != len(want) {
		t.Fatalf("message count: got %d, want %d", len(got), len(want))
	}
	for i, m := range got {
		if m.Role != want[i].Role {
			t.Errorf("msg[%d].Role: got %q, want %q", i, m.Role, want[i].Role)
		}
		if m.Content != want[i].Content {
			t.Errorf("msg[%d].Content: got %q, want %q", i, m.Content, want[i].Content)
		}
	}
}

func TestLoadNonExistentSession(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	msgs, meta, err := store.Load("does-not-exist")
	if err != nil {
		t.Fatalf("Load non-existent: expected no error, got %v", err)
	}
	if msgs != nil {
		t.Errorf("expected nil messages, got %v", msgs)
	}
	if meta != nil {
		t.Errorf("expected nil meta, got %v", meta)
	}
}

func TestMultipleSessionsNoInterference(t *testing.T) {
	dir := t.TempDir()
	store := New(dir)

	// Write to session A.
	if err := store.Append("sess-a", llm.Message{Role: "user", Content: "alpha"}); err != nil {
		t.Fatalf("Append sess-a: %v", err)
	}
	if err := store.WriteMeta("sess-a", map[string]string{"agent": "a"}); err != nil {
		t.Fatalf("WriteMeta sess-a: %v", err)
	}

	// Write to session B.
	if err := store.Append("sess-b", llm.Message{Role: "user", Content: "beta"}); err != nil {
		t.Fatalf("Append sess-b: %v", err)
	}
	if err := store.WriteMeta("sess-b", map[string]string{"agent": "b"}); err != nil {
		t.Fatalf("WriteMeta sess-b: %v", err)
	}

	// Load A and verify isolation.
	msgsA, metaA, err := store.Load("sess-a")
	if err != nil {
		t.Fatalf("Load sess-a: %v", err)
	}
	if len(msgsA) != 1 || msgsA[0].Content != "alpha" {
		t.Errorf("sess-a messages: got %v, want [{user alpha}]", msgsA)
	}
	if metaA["agent"] != "a" {
		t.Errorf("sess-a meta[agent]: got %q, want %q", metaA["agent"], "a")
	}

	// Load B and verify isolation.
	msgsB, metaB, err := store.Load("sess-b")
	if err != nil {
		t.Fatalf("Load sess-b: %v", err)
	}
	if len(msgsB) != 1 || msgsB[0].Content != "beta" {
		t.Errorf("sess-b messages: got %v, want [{user beta}]", msgsB)
	}
	if metaB["agent"] != "b" {
		t.Errorf("sess-b meta[agent]: got %q, want %q", metaB["agent"], "b")
	}

	// Verify separate files exist.
	if _, err := os.Stat(filepath.Join(dir, "sess-a.jsonl")); err != nil {
		t.Errorf("sess-a.jsonl should exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sess-b.jsonl")); err != nil {
		t.Errorf("sess-b.jsonl should exist: %v", err)
	}
}
