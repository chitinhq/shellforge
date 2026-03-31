package llm

import (
	"os"

	"github.com/AgentGuardHQ/shellforge/internal/ollama"
)

// OllamaProvider wraps the existing ollama.Chat() function.
type OllamaProvider struct {
	host  string
	model string
}

// NewOllamaProvider creates an OllamaProvider targeting the given host and model.
// Pass empty strings to use the ollama package defaults.
func NewOllamaProvider(host, model string) *OllamaProvider {
	return &OllamaProvider{host: host, model: model}
}

// Name returns the provider identifier.
func (o *OllamaProvider) Name() string {
	return "ollama"
}

// Chat converts llm.Message slice to ollama.ChatMessage, calls ollama.Chat(),
// and converts the response back to llm.Response.
// Ollama does not support native tool-use, so ToolCalls is always nil.
// The tools parameter is ignored — Ollama uses text-based tool calling via prompt.
// Roles: "tool_result" is mapped to "user" since Ollama only understands "user".
func (o *OllamaProvider) Chat(messages []Message, tools []ToolDef) (*Response, error) {
	// Override ollama package host if caller specified one.
	if o.host != "" {
		prev := ollama.Host
		ollama.Host = o.host
		defer func() { ollama.Host = prev }()
	}

	ollamaMsgs := make([]ollama.ChatMessage, len(messages))
	for i, m := range messages {
		role := m.Role
		if role == "tool_result" {
			role = "user"
		}
		ollamaMsgs[i] = ollama.ChatMessage{
			Role:    role,
			Content: m.Content,
		}
	}

	resp, err := ollama.Chat(ollamaMsgs, o.model)
	if err != nil {
		return nil, err
	}

	return &Response{
		Content:    resp.Message.Content,
		ToolCalls:  nil,
		StopReason: "end_turn",
		PromptTok:  resp.PromptEval,
		OutputTok:  resp.EvalCount,
	}, nil
}

// envOr returns the value of the environment variable key, or fallback if unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
