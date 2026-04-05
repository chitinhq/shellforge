package llm

import "testing"

func TestSingleCompleteEvent(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data: hello world\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event, got none")
	}
	if ev.Type != "message" {
		t.Errorf("Type: got %q, want %q", ev.Type, "message")
	}
	if ev.Data != "hello world" {
		t.Errorf("Data: got %q, want %q", ev.Data, "hello world")
	}
}

func TestPartialPacketReassembly(t *testing.T) {
	s := NewEventStream()

	// Push the event in three fragments.
	s.Push([]byte("data: hel"))
	if _, ok := s.Next(); ok {
		t.Fatal("should not have event yet after first fragment")
	}

	s.Push([]byte("lo wor"))
	if _, ok := s.Next(); ok {
		t.Fatal("should not have event yet after second fragment")
	}

	s.Push([]byte("ld\n\n"))
	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event after final fragment")
	}
	if ev.Data != "hello world" {
		t.Errorf("Data: got %q, want %q", ev.Data, "hello world")
	}
}

func TestMultipleEventsInSinglePush(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data: first\n\ndata: second\n\ndata: third\n\n"))

	want := []string{"first", "second", "third"}
	for i, w := range want {
		ev, ok := s.Next()
		if !ok {
			t.Fatalf("event %d: expected event, got none", i)
		}
		if ev.Data != w {
			t.Errorf("event %d: Data: got %q, want %q", i, ev.Data, w)
		}
	}

	if _, ok := s.Next(); ok {
		t.Error("expected no more events")
	}
}

func TestMultipleDataLines(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data: line one\ndata: line two\ndata: line three\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	want := "line one\nline two\nline three"
	if ev.Data != want {
		t.Errorf("Data: got %q, want %q", ev.Data, want)
	}
}

func TestCommentsIgnored(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte(":ping\ndata: real data\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Data != "real data" {
		t.Errorf("Data: got %q, want %q", ev.Data, "real data")
	}
}

func TestCommentOnlyFrameSkipped(t *testing.T) {
	s := NewEventStream()
	// Comment-only frame followed by a real event.
	s.Push([]byte(":ping\n\ndata: after comment\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event after comment-only frame")
	}
	if ev.Data != "after comment" {
		t.Errorf("Data: got %q, want %q", ev.Data, "after comment")
	}
}

func TestDoneMarkerIgnored(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data: real\n\ndata: [DONE]\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected first event")
	}
	if ev.Data != "real" {
		t.Errorf("Data: got %q, want %q", ev.Data, "real")
	}

	// The [DONE] frame should be skipped (no data fields remain after filtering).
	if _, ok := s.Next(); ok {
		t.Error("expected [DONE] frame to be skipped")
	}
}

func TestCustomEventType(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("event: content_block_delta\ndata: {\"type\":\"text_delta\"}\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Type != "content_block_delta" {
		t.Errorf("Type: got %q, want %q", ev.Type, "content_block_delta")
	}
	if ev.Data != `{"type":"text_delta"}` {
		t.Errorf("Data: got %q, want %q", ev.Data, `{"type":"text_delta"}`)
	}
}

func TestEmptyDataField(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data:\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event with empty data")
	}
	if ev.Data != "" {
		t.Errorf("Data: got %q, want %q", ev.Data, "")
	}
}

func TestEventWithID(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("id: evt_123\ndata: payload\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.ID != "evt_123" {
		t.Errorf("ID: got %q, want %q", ev.ID, "evt_123")
	}
	if ev.Data != "payload" {
		t.Errorf("Data: got %q, want %q", ev.Data, "payload")
	}
}

func TestCRLFLineEndings(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data: crlf test\r\n\r\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event with CRLF endings")
	}
	if ev.Data != "crlf test" {
		t.Errorf("Data: got %q, want %q", ev.Data, "crlf test")
	}
}

func TestMixedLineEndings(t *testing.T) {
	s := NewEventStream()
	// First event uses \r\n, second uses \n.
	s.Push([]byte("data: first\r\n\r\ndata: second\n\n"))

	ev1, ok := s.Next()
	if !ok {
		t.Fatal("expected first event")
	}
	if ev1.Data != "first" {
		t.Errorf("first Data: got %q, want %q", ev1.Data, "first")
	}

	ev2, ok := s.Next()
	if !ok {
		t.Fatal("expected second event")
	}
	if ev2.Data != "second" {
		t.Errorf("second Data: got %q, want %q", ev2.Data, "second")
	}
}

func TestNoEventsReturnsFalse(t *testing.T) {
	s := NewEventStream()
	if _, ok := s.Next(); ok {
		t.Error("expected false on empty stream")
	}

	// Partial data, no frame boundary yet.
	s.Push([]byte("data: incomplete"))
	if _, ok := s.Next(); ok {
		t.Error("expected false on incomplete frame")
	}
}

func TestRetryFieldIgnored(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("retry: 3000\ndata: with retry\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Data != "with retry" {
		t.Errorf("Data: got %q, want %q", ev.Data, "with retry")
	}
}

func TestDataWithNoSpaceAfterColon(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data:nospace\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Data != "nospace" {
		t.Errorf("Data: got %q, want %q", ev.Data, "nospace")
	}
}

func TestStreamEventTypes(t *testing.T) {
	// Verify StreamEvent struct fields are accessible.
	tc := ToolCall{ID: "tc_1", Name: "bash"}
	se := StreamEvent{
		Type:     "tool_use_start",
		ToolCall: &tc,
	}
	if se.Type != "tool_use_start" {
		t.Errorf("Type: got %q, want %q", se.Type, "tool_use_start")
	}
	if se.ToolCall.Name != "bash" {
		t.Errorf("ToolCall.Name: got %q, want %q", se.ToolCall.Name, "bash")
	}

	se2 := StreamEvent{
		Type:  "text_delta",
		Text:  "hello",
	}
	if se2.Text != "hello" {
		t.Errorf("Text: got %q, want %q", se2.Text, "hello")
	}

	se3 := StreamEvent{
		Type:  "error",
		Error: "rate limited",
	}
	if se3.Error != "rate limited" {
		t.Errorf("Error: got %q, want %q", se3.Error, "rate limited")
	}
}

func TestDefaultEventType(t *testing.T) {
	s := NewEventStream()
	// No "event:" field — type should default to "message".
	s.Push([]byte("data: no explicit type\n\n"))

	ev, ok := s.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.Type != "message" {
		t.Errorf("Type: got %q, want %q", ev.Type, "message")
	}
}

func TestBufferDrainedAfterConsumption(t *testing.T) {
	s := NewEventStream()
	s.Push([]byte("data: one\n\ndata: two\n\n"))

	s.Next()
	s.Next()

	// Buffer should be empty — no more events.
	if _, ok := s.Next(); ok {
		t.Error("expected no events after draining")
	}
}
