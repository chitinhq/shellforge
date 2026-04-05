package llm

import "bytes"

// Event is a parsed SSE event.
type Event struct {
	Type string // from "event:" field, defaults to "message"
	Data string // from "data:" field(s), joined with newlines
	ID   string // from "id:" field
}

// StreamEvent represents a typed event from a streaming LLM response.
type StreamEvent struct {
	Type     string    // "text_delta", "tool_use_start", "tool_use_delta", "tool_use_end", "message_stop", "thinking_delta", "error"
	Text     string    // for text_delta and thinking_delta
	ToolCall *ToolCall // for tool_use_start (ID and Name populated)
	ToolJSON string    // for tool_use_delta (partial JSON arguments)
	Error    string    // for error events
}

// EventStream is a buffered SSE parser. It accepts raw bytes from an HTTP
// response body and extracts complete SSE frames, handling partial packets,
// multiple line-ending conventions, comments, and [DONE] markers.
type EventStream struct {
	buf []byte
}

// NewEventStream creates a new SSE parser.
func NewEventStream() *EventStream {
	return &EventStream{}
}

// Push adds raw bytes from the HTTP response body to the buffer.
func (s *EventStream) Push(chunk []byte) {
	s.buf = append(s.buf, chunk...)
}

// Next extracts the next complete SSE event from the buffer.
// Returns the event and true if one was available, or zero Event and false.
func (s *EventStream) Next() (Event, bool) {
	for {
		// Look for the earliest frame boundary: a blank line separating events.
		// SSE frames are terminated by two consecutive line endings.
		idx := findFrameBoundary(s.buf)
		if idx < 0 {
			return Event{}, false
		}

		// Extract the raw frame and advance the buffer past the boundary.
		frame, rest := splitAtBoundary(s.buf, idx)
		s.buf = rest

		// Parse the frame into an event. If the frame contained only comments
		// or a [DONE] marker, ev will be zero-valued and ok will be false.
		ev, ok := parseFrame(frame)
		if ok {
			return ev, true
		}
		// Frame was empty/comments-only/[DONE] — try the next frame.
	}
}

// findFrameBoundary returns the index of the first blank-line boundary in buf,
// or -1 if no complete frame is available. Handles \n\n and \r\n\r\n.
func findFrameBoundary(buf []byte) int {
	for i := 0; i < len(buf)-1; i++ {
		if buf[i] == '\n' && buf[i+1] == '\n' {
			return i
		}
		if buf[i] == '\n' && i+2 < len(buf) && buf[i+1] == '\r' && buf[i+2] == '\n' {
			return i
		}
		if buf[i] == '\r' && i+3 < len(buf) && buf[i+1] == '\n' && buf[i+2] == '\r' && buf[i+3] == '\n' {
			return i
		}
	}
	return -1
}

// splitAtBoundary splits buf at the frame boundary index, returning the frame
// content and the remaining buffer after the boundary.
func splitAtBoundary(buf []byte, idx int) (frame, rest []byte) {
	frame = buf[:idx]

	// Advance past the boundary characters.
	i := idx
	for i < len(buf) && (buf[i] == '\n' || buf[i] == '\r') {
		i++
	}
	return frame, buf[i:]
}

// parseFrame parses a single SSE frame (lines between blank-line boundaries)
// into an Event. Returns false if the frame has no meaningful data (e.g., all
// comments, empty, or [DONE]).
func parseFrame(frame []byte) (Event, bool) {
	var ev Event
	hasData := false
	var dataLines [][]byte

	lines := splitLines(frame)
	for _, line := range lines {
		// Skip empty lines within a frame.
		if len(line) == 0 {
			continue
		}
		// SSE comments start with ':'.
		if line[0] == ':' {
			continue
		}

		field, value := parseField(line)
		switch field {
		case "event":
			ev.Type = value
		case "data":
			// Check for [DONE] marker.
			if value == "[DONE]" {
				continue
			}
			dataLines = append(dataLines, []byte(value))
			hasData = true
		case "id":
			ev.ID = value
		case "retry":
			// Acknowledged but not stored — no reconnection logic needed.
		}
	}

	if !hasData && ev.Type == "" && ev.ID == "" {
		return Event{}, false
	}

	// Default event type per SSE spec.
	if ev.Type == "" {
		ev.Type = "message"
	}

	// Join multiple data: lines with newlines per SSE spec.
	ev.Data = string(bytes.Join(dataLines, []byte("\n")))

	return ev, true
}

// splitLines splits a frame into individual lines, handling both \n and \r\n.
func splitLines(frame []byte) [][]byte {
	var lines [][]byte
	for len(frame) > 0 {
		idx := bytes.IndexByte(frame, '\n')
		if idx < 0 {
			// Last line without trailing newline.
			line := frame
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			break
		}
		line := frame[:idx]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
		frame = frame[idx+1:]
	}
	return lines
}

// parseField splits an SSE line into field name and value.
// "data: hello" -> ("data", "hello")
// "data:hello"  -> ("data", "hello")
func parseField(line []byte) (field, value string) {
	idx := bytes.IndexByte(line, ':')
	if idx < 0 {
		return string(line), ""
	}
	field = string(line[:idx])
	val := line[idx+1:]
	// Strip optional leading space after colon per SSE spec.
	if len(val) > 0 && val[0] == ' ' {
		val = val[1:]
	}
	return field, string(val)
}
