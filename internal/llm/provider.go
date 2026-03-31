package llm

// Provider abstracts an LLM backend (Ollama, Anthropic, etc.).
type Provider interface {
	Chat(messages []Message, tools []ToolDef) (*Response, error)
	Name() string
}

// Message is a conversation turn.
type Message struct {
	Role       string // "system", "user", "assistant", "tool_result"
	Content    string
	ToolCallID string // set when Role == "tool_result"
}

// ToolDef describes a tool the model can invoke.
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema
}

// ToolCall is a model's request to invoke a tool.
type ToolCall struct {
	ID     string
	Name   string
	Params map[string]string
}

// Response is the model's reply to a Chat call.
type Response struct {
	Content    string
	ToolCalls  []ToolCall
	StopReason string // "end_turn", "tool_use"
	PromptTok  int
	OutputTok  int
}
