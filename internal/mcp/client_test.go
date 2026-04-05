package mcp

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// mockPipe wires an io.Pipe into the shapes expected by newTestClient.
// serverReader is what the "server" reads (client's stdin).
// serverWriter is what the "server" writes (client's stdout).
func mockPipe() (clientStdin io.WriteCloser, clientStdout io.ReadCloser, serverReader io.ReadCloser, serverWriter io.WriteCloser) {
	sr, cw := io.Pipe() // client writes -> server reads
	cr, sw := io.Pipe() // server writes -> client reads
	return cw, cr, sr, sw
}

// ── send tests ──

func TestSend_WritesValidJSONRPC(t *testing.T) {
	clientStdin, clientStdout, serverReader, serverWriter := mockPipe()
	c := newTestClient(clientStdin, clientStdout)

	// Server goroutine: read the request, write a response.
	go func() {
		reader := bufio.NewReader(serverReader)
		line, _ := reader.ReadBytes('\n')

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			t.Errorf("server: unmarshal request: %v", err)
			return
		}
		if req.JSONRPC != "2.0" {
			t.Errorf("expected jsonrpc 2.0, got %q", req.JSONRPC)
		}
		if req.Method != "test/method" {
			t.Errorf("expected method test/method, got %q", req.Method)
		}
		if req.ID != 1 {
			t.Errorf("expected id 1, got %d", req.ID)
		}

		resp := `{"jsonrpc":"2.0","id":1,"result":{"ok":true}}` + "\n"
		serverWriter.Write([]byte(resp))
	}()

	resp, err := c.send("test/method", map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("send returned error: %v", err)
	}
	if resp.ID != 1 {
		t.Fatalf("expected response id 1, got %d", resp.ID)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error in response: %v", resp.Error)
	}
}

func TestSend_IncrementsID(t *testing.T) {
	clientStdin, clientStdout, serverReader, serverWriter := mockPipe()
	c := newTestClient(clientStdin, clientStdout)

	go func() {
		reader := bufio.NewReader(serverReader)
		for i := 1; i <= 3; i++ {
			reader.ReadBytes('\n')
			resp, _ := json.Marshal(jsonRPCResponse{JSONRPC: "2.0", ID: i})
			serverWriter.Write(append(resp, '\n'))
		}
	}()

	for expected := 1; expected <= 3; expected++ {
		resp, err := c.send("ping", nil)
		if err != nil {
			t.Fatalf("send %d returned error: %v", expected, err)
		}
		if resp.ID != expected {
			t.Fatalf("expected response id %d, got %d", expected, resp.ID)
		}
	}
}

// ── readResponse tests ──

func TestReadResponse_ParsesValidJSON(t *testing.T) {
	_, clientStdout, _, serverWriter := mockPipe()
	c := &Client{
		stdout: bufio.NewReader(clientStdout),
	}

	go func() {
		resp := `{"jsonrpc":"2.0","id":42,"result":{"tools":[]}}` + "\n"
		serverWriter.Write([]byte(resp))
	}()

	resp, err := c.readResponse()
	if err != nil {
		t.Fatalf("readResponse returned error: %v", err)
	}
	if resp.ID != 42 {
		t.Fatalf("expected id 42, got %d", resp.ID)
	}
	if resp.Error != nil {
		t.Fatal("expected no error in response")
	}
}

func TestReadResponse_ParsesErrorResponse(t *testing.T) {
	_, clientStdout, _, serverWriter := mockPipe()
	c := &Client{
		stdout: bufio.NewReader(clientStdout),
	}

	go func() {
		resp := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"method not found"}}` + "\n"
		serverWriter.Write([]byte(resp))
	}()

	resp, err := c.readResponse()
	if err != nil {
		t.Fatalf("readResponse returned error: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
	if resp.Error.Code != -32601 {
		t.Fatalf("expected error code -32601, got %d", resp.Error.Code)
	}
	if resp.Error.Message != "method not found" {
		t.Fatalf("expected 'method not found', got %q", resp.Error.Message)
	}
}

func TestReadResponse_ReturnsErrorOnMalformedJSON(t *testing.T) {
	_, clientStdout, _, serverWriter := mockPipe()
	c := &Client{
		stdout: bufio.NewReader(clientStdout),
	}

	go func() {
		serverWriter.Write([]byte("this is not json\n"))
	}()

	_, err := c.readResponse()
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected 'decode response' in error, got: %v", err)
	}
}

// ── CallTool tests ──

func TestCallTool_SendsCorrectParams(t *testing.T) {
	clientStdin, clientStdout, serverReader, serverWriter := mockPipe()
	c := newTestClient(clientStdin, clientStdout)
	c.toolIndex["test_tool"] = Tool{
		Name:        "test_tool",
		Description: "a test tool",
	}

	go func() {
		reader := bufio.NewReader(serverReader)
		line, _ := reader.ReadBytes('\n')

		var req jsonRPCRequest
		json.Unmarshal(line, &req)

		if req.Method != "tools/call" {
			t.Errorf("expected method tools/call, got %q", req.Method)
		}

		// Verify params contain tool name and arguments.
		params, _ := json.Marshal(req.Params)
		var p map[string]interface{}
		json.Unmarshal(params, &p)

		if p["name"] != "test_tool" {
			t.Errorf("expected tool name 'test_tool', got %v", p["name"])
		}
		args := p["arguments"].(map[string]interface{})
		if args["input"] != "hello" {
			t.Errorf("expected argument input=hello, got %v", args["input"])
		}

		resp := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"result text"}]}}` + "\n"
		serverWriter.Write([]byte(resp))
	}()

	result, err := c.CallTool("test_tool", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if result != "result text" {
		t.Fatalf("expected 'result text', got %q", result)
	}
}

func TestCallTool_ReturnsErrorOnJSONRPCError(t *testing.T) {
	clientStdin, clientStdout, serverReader, serverWriter := mockPipe()
	c := newTestClient(clientStdin, clientStdout)
	c.toolIndex["fail_tool"] = Tool{Name: "fail_tool"}

	go func() {
		reader := bufio.NewReader(serverReader)
		reader.ReadBytes('\n')
		resp := `{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"tool execution failed"}}` + "\n"
		serverWriter.Write([]byte(resp))
	}()

	_, err := c.CallTool("fail_tool", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error from CallTool")
	}
	if !strings.Contains(err.Error(), "tool execution failed") {
		t.Fatalf("expected 'tool execution failed' in error, got: %v", err)
	}
}

func TestCallTool_ReturnsErrorForUnknownTool(t *testing.T) {
	clientStdin, clientStdout, _, _ := mockPipe()
	c := newTestClient(clientStdin, clientStdout)

	_, err := c.CallTool("nonexistent", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Fatalf("expected 'unknown tool' in error, got: %v", err)
	}
}

func TestCallTool_ConcatenatesMultipleTextBlocks(t *testing.T) {
	clientStdin, clientStdout, serverReader, serverWriter := mockPipe()
	c := newTestClient(clientStdin, clientStdout)
	c.toolIndex["multi"] = Tool{Name: "multi"}

	go func() {
		reader := bufio.NewReader(serverReader)
		reader.ReadBytes('\n')
		resp := `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"part1"},{"type":"text","text":"part2"}]}}` + "\n"
		serverWriter.Write([]byte(resp))
	}()

	result, err := c.CallTool("multi", map[string]interface{}{})
	if err != nil {
		t.Fatalf("CallTool returned error: %v", err)
	}
	if result != "part1part2" {
		t.Fatalf("expected 'part1part2', got %q", result)
	}
}

// ── Tools tests ──

func TestTools_ReturnsDiscoveredTools(t *testing.T) {
	clientStdin, clientStdout, _, _ := mockPipe()
	c := newTestClient(clientStdin, clientStdout)
	c.toolIndex["tool_a"] = Tool{Name: "tool_a", Description: "Tool A"}
	c.toolIndex["tool_b"] = Tool{Name: "tool_b", Description: "Tool B"}

	tools := c.Tools()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	found := make(map[string]bool)
	for _, tool := range tools {
		found[tool.Name] = true
	}
	if !found["tool_a"] || !found["tool_b"] {
		t.Fatalf("expected tool_a and tool_b, got %v", tools)
	}
}

func TestTools_ReturnsEmptyWhenNoTools(t *testing.T) {
	clientStdin, clientStdout, _, _ := mockPipe()
	c := newTestClient(clientStdin, clientStdout)

	tools := c.Tools()
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(tools))
	}
}

// ── Close tests ──

func TestClose_SafeToCallMultipleTimes(t *testing.T) {
	clientStdin, clientStdout, _, _ := mockPipe()
	c := newTestClient(clientStdin, clientStdout)

	// First close should succeed.
	if err := c.Close(); err != nil {
		t.Fatalf("first Close returned error: %v", err)
	}

	// Second close should not panic or return a fatal error.
	// stdin is already closed, so we ignore the error from a double-close.
	c.Close()
}

func TestClose_NilFieldsSafe(t *testing.T) {
	c := &Client{
		toolIndex: make(map[string]Tool),
	}

	// Close with nil stdin and cmd should not panic.
	if err := c.Close(); err != nil {
		t.Fatalf("Close with nil fields returned error: %v", err)
	}
}
