package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider calls an OpenAI-compatible chat completions API (e.g. DeepSeek).
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIProvider creates an OpenAIProvider.
func NewOpenAIProvider(apiKey, baseURL, model string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

// Name returns the provider identifier.
func (o *OpenAIProvider) Name() string {
	return "openai"
}

// ----------------------------------------------------------------------------
// OpenAI API wire types
// ----------------------------------------------------------------------------

// openAIMessage is one turn in the OpenAI messages array.
type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

// openAIToolCall is a tool call in an assistant message.
type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIToolFunction `json:"function"`
}

// openAIToolFunction holds the name and JSON-encoded arguments.
type openAIToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// openAITool is a tool definition in the OpenAI format.
type openAITool struct {
	Type     string             `json:"type"`
	Function openAIToolDef      `json:"function"`
}

// openAIToolDef is the function portion of a tool definition.
type openAIToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// openAIRequest is the full POST body for /chat/completions.
type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
	Tools    []openAITool    `json:"tools,omitempty"`
}

// openAIResponse is the API response body.
type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// openAIChoice is one completion choice.
type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// openAIErrorResponse represents an API error body.
type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

// ----------------------------------------------------------------------------
// Chat implementation
// ----------------------------------------------------------------------------

// Chat sends messages to the OpenAI-compatible API and returns the response.
func (o *OpenAIProvider) Chat(messages []Message, tools []ToolDef) (*Response, error) {
	// Convert messages to OpenAI wire format.
	apiMsgs := make([]openAIMessage, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "tool_result":
			// tool_result maps to role "tool" with tool_call_id.
			apiMsgs = append(apiMsgs, openAIMessage{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})

		case "assistant":
			msg := openAIMessage{
				Role:    "assistant",
				Content: m.Content,
			}
			if len(m.ToolCalls) > 0 {
				// Assistant message with tool calls — include tool_calls array.
				msg.ToolCalls = make([]openAIToolCall, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					argsJSON, err := json.Marshal(tc.Params)
					if err != nil {
						return nil, fmt.Errorf("openai: marshal tool call params: %w", err)
					}
					msg.ToolCalls[i] = openAIToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: openAIToolFunction{
							Name:      tc.Name,
							Arguments: string(argsJSON),
						},
					}
				}
			}
			apiMsgs = append(apiMsgs, msg)

		default:
			// system, user pass through as-is.
			apiMsgs = append(apiMsgs, openAIMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	// Convert tool definitions.
	var apiTools []openAITool
	for _, t := range tools {
		schema := t.Parameters
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		apiTools = append(apiTools, openAITool{
			Type: "function",
			Function: openAIToolDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		})
	}

	reqBody := openAIRequest{
		Model:    o.model,
		Messages: apiMsgs,
		Tools:    apiTools,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, o.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr openAIErrorResponse
		if jsonErr := json.Unmarshal(respBytes, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("openai: api error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("openai: http %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("openai: unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("openai: no choices in response")
	}

	choice := apiResp.Choices[0]
	result := &Response{
		Content:   choice.Message.Content,
		PromptTok: apiResp.Usage.PromptTokens,
		OutputTok: apiResp.Usage.CompletionTokens,
		// CacheCreated and CacheHit are not supported by OpenAI-compatible APIs.
	}

	// Map finish_reason to StopReason.
	switch choice.FinishReason {
	case "tool_calls":
		result.StopReason = "tool_use"
	default:
		result.StopReason = "end_turn"
	}

	// Parse tool calls if present.
	if len(choice.Message.ToolCalls) > 0 {
		for _, tc := range choice.Message.ToolCalls {
			params := make(map[string]string)
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
				// Degrade gracefully — bad JSON in arguments should not crash the loop.
				result.StopReason = "end_turn"
				result.ToolCalls = nil
				if result.Content == "" {
					result.Content = "Tool call failed to parse"
				}
				return result, nil
			}
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:     tc.ID,
				Name:   tc.Function.Name,
				Params: params,
			})
		}
	}

	return result, nil
}
