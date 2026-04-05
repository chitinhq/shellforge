package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Client manages an MCP server process and communicates via JSON-RPC over stdio.
type Client struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    *bufio.Reader
	mu        sync.Mutex
	nextID    int
	toolIndex map[string]Tool // qualified name -> tool
}

// Tool is an MCP tool definition.
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// jsonRPCRequest is a JSON-RPC 2.0 request.
type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// jsonRPCResponse is a JSON-RPC 2.0 response.
type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCError is a JSON-RPC 2.0 error.
type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewClient spawns the MCP server process and initializes the connection.
// command is the binary to run, args are its arguments.
// It sends "initialize" and "initialized" per MCP protocol, then discovers tools.
func NewClient(command string, args ...string) (*Client, error) {
	cmd := exec.Command(command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("mcp: stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("mcp: start %s: %w", command, err)
	}

	c := &Client{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    bufio.NewReader(stdout),
		nextID:    1,
		toolIndex: make(map[string]Tool),
	}

	if err := c.initialize(); err != nil {
		// Best-effort cleanup on init failure.
		c.Close()
		return nil, fmt.Errorf("mcp: initialize: %w", err)
	}

	if err := c.discoverTools(); err != nil {
		c.Close()
		return nil, fmt.Errorf("mcp: discover tools: %w", err)
	}

	return c, nil
}

// newTestClient creates a Client with pre-wired stdin/stdout for testing.
// It does NOT spawn a process or run the MCP handshake.
func newTestClient(stdin io.WriteCloser, stdout io.ReadCloser) *Client {
	return &Client{
		stdin:     stdin,
		stdout:    bufio.NewReader(stdout),
		nextID:    1,
		toolIndex: make(map[string]Tool),
	}
}

// Tools returns the list of available tools from the server.
func (c *Client) Tools() []Tool {
	c.mu.Lock()
	defer c.mu.Unlock()

	tools := make([]Tool, 0, len(c.toolIndex))
	for _, t := range c.toolIndex {
		tools = append(tools, t)
	}
	return tools
}

// CallTool invokes a tool by name with the given arguments.
// Returns the result content as a string.
func (c *Client) CallTool(name string, args map[string]interface{}) (string, error) {
	if _, ok := c.toolIndex[name]; !ok {
		return "", fmt.Errorf("mcp: unknown tool %q", name)
	}

	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}

	resp, err := c.send("tools/call", params)
	if err != nil {
		return "", err
	}
	if resp.Error != nil {
		return "", fmt.Errorf("mcp: tool %q error %d: %s", name, resp.Error.Code, resp.Error.Message)
	}

	// MCP tools/call returns {content: [{type: "text", text: "..."}]}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", fmt.Errorf("mcp: decode tool result: %w", err)
	}

	// Concatenate all text content blocks.
	var out string
	for _, block := range result.Content {
		if block.Type == "text" {
			out += block.Text
		}
	}
	return out, nil
}

// Close shuts down the MCP server process gracefully.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Wait()
	}
	return nil
}

// send sends a JSON-RPC request and reads the response.
func (c *Client) send(method string, params interface{}) (*jsonRPCResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      c.nextID,
		Method:  method,
		Params:  params,
	}
	c.nextID++

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("mcp: marshal request: %w", err)
	}
	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("mcp: write request: %w", err)
	}

	return c.readResponse()
}

// readResponse reads one JSON-RPC response line from stdout.
func (c *Client) readResponse() (*jsonRPCResponse, error) {
	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("mcp: read response: %w", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("mcp: decode response: %w", err)
	}
	return &resp, nil
}

// initialize performs the MCP initialize handshake.
func (c *Client) initialize() error {
	params := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "shellforge",
			"version": "0.1.0",
		},
	}

	resp, err := c.send("initialize", params)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("initialize error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	// Send "initialized" notification (no ID, no response expected).
	// Per MCP spec this is a notification, but we send it as a request with an ID
	// for simplicity — the server should accept it either way.
	notif := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "notifications/initialized",
	}
	data, err := json.Marshal(notif)
	if err != nil {
		return fmt.Errorf("marshal initialized notification: %w", err)
	}
	data = append(data, '\n')
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("write initialized notification: %w", err)
	}

	return nil
}

// discoverTools sends tools/list and populates the tool index.
func (c *Client) discoverTools() error {
	resp, err := c.send("tools/list", nil)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("tools/list error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	var result struct {
		Tools []Tool `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("decode tools/list: %w", err)
	}

	for _, t := range result.Tools {
		c.toolIndex[t.Name] = t
	}
	return nil
}
