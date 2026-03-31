package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const (
	defaultAnthropicModel   = "claude-haiku-4-5-20251001"
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicVersion        = "2023-06-01"
	anthropicMaxTokens      = 4096
)

// AnthropicProvider calls the Anthropic Messages API via stdlib HTTP.
type AnthropicProvider struct {
	apiKey         string
	model          string
	baseURL        string
	client         *http.Client
	ThinkingBudget int // max thinking tokens (0 = disabled)
}

// NewAnthropicProvider creates an AnthropicProvider.
// Pass empty strings to fall back to ANTHROPIC_API_KEY and ANTHROPIC_MODEL env vars.
func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if model == "" {
		model = envOr("ANTHROPIC_MODEL", defaultAnthropicModel)
	}
	return &AnthropicProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultAnthropicBaseURL,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

// Name returns the provider identifier.
func (a *AnthropicProvider) Name() string {
	return "anthropic"
}

// ----------------------------------------------------------------------------
// Anthropic API wire types
// ----------------------------------------------------------------------------

// anthropicContentBlock is a polymorphic content block in the Anthropic API.
type anthropicContentBlock struct {
	Type       string         `json:"type"`
	Text       string         `json:"text,omitempty"`
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Input      map[string]any `json:"input,omitempty"`
	ToolUseID  string         `json:"tool_use_id,omitempty"`
	Content    string         `json:"content,omitempty"`
}

// anthropicMessage is one turn in the Anthropic messages array.
// Content can be a plain string or an array of content blocks.
// We use json.RawMessage to handle both.
type anthropicMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// cacheControl instructs Anthropic to cache this content block for 5 minutes.
type cacheControl struct {
	Type string `json:"type"`
}

// anthropicToolDef is the Anthropic tool definition format.
type anthropicToolDef struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	InputSchema  map[string]any `json:"input_schema"`
	CacheControl *cacheControl  `json:"cache_control,omitempty"`
}

// anthropicThinking configures extended thinking (chain-of-thought).
type anthropicThinking struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // max thinking tokens
}

// anthropicRequest is the full POST body for /v1/messages.
type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    json.RawMessage     `json:"system,omitempty"`
	Messages  []anthropicMessage  `json:"messages"`
	Tools     []anthropicToolDef  `json:"tools,omitempty"`
	Thinking  *anthropicThinking  `json:"thinking,omitempty"`
}

// anthropicResponse is the API response body.
type anthropicResponse struct {
	ID         string                  `json:"id"`
	Content    []anthropicContentBlock `json:"content"`
	StopReason string                  `json:"stop_reason"`
	Usage      struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	} `json:"usage"`
}

// anthropicErrorResponse represents an API error body.
type anthropicErrorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// ----------------------------------------------------------------------------
// Chat implementation
// ----------------------------------------------------------------------------

// Chat sends messages to the Anthropic Messages API and returns the response.
func (a *AnthropicProvider) Chat(messages []Message, tools []ToolDef) (*Response, error) {
	// Separate system messages from conversation messages.
	var systemPrompt string
	var convMsgs []Message
	for _, m := range messages {
		if m.Role == "system" {
			if systemPrompt != "" {
				systemPrompt += "\n"
			}
			systemPrompt += m.Content
		} else {
			convMsgs = append(convMsgs, m)
		}
	}

	// Convert conversation messages to Anthropic wire format.
	apiMsgs, err := convertMessages(convMsgs)
	if err != nil {
		return nil, fmt.Errorf("anthropic: convert messages: %w", err)
	}

	// Convert tool definitions.
	apiTools := make([]anthropicToolDef, len(tools))
	for i, t := range tools {
		schema := t.Parameters
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		apiTools[i] = anthropicToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		}
	}

	reqBody := anthropicRequest{
		Model:     a.model,
		MaxTokens: anthropicMaxTokens,
		Messages:  apiMsgs,
	}

	// Enable extended thinking with budget cap if configured.
	if a.ThinkingBudget > 0 {
		reqBody.Thinking = &anthropicThinking{
			Type:         "enabled",
			BudgetTokens: a.ThinkingBudget,
		}
		// When thinking is enabled, max_tokens must cover thinking + output.
		if reqBody.MaxTokens < a.ThinkingBudget+1024 {
			reqBody.MaxTokens = a.ThinkingBudget + 1024
		}
	}

	// Build system as array of content blocks with cache_control on the last block.
	if systemPrompt != "" {
		systemBlocks := []map[string]any{
			{
				"type":          "text",
				"text":          systemPrompt,
				"cache_control": map[string]string{"type": "ephemeral"},
			},
		}
		reqBody.System, _ = json.Marshal(systemBlocks)
	}

	// Add cache_control to last tool so tool definitions are cached.
	if len(apiTools) > 0 {
		apiTools[len(apiTools)-1].CacheControl = &cacheControl{Type: "ephemeral"}
		reqBody.Tools = apiTools
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, a.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr anthropicErrorResponse
		if jsonErr := json.Unmarshal(respBytes, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("anthropic: api error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("anthropic: http %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("anthropic: unmarshal response: %w", err)
	}

	return parseResponse(&apiResp), nil
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// convertMessages converts llm.Message slice to Anthropic API message slice.
func convertMessages(messages []Message) ([]anthropicMessage, error) {
	result := make([]anthropicMessage, 0, len(messages))
	for _, m := range messages {
		var raw json.RawMessage
		var err error

		switch m.Role {
		case "tool_result":
			// Must be wrapped in a user message with tool_result content block.
			block := anthropicContentBlock{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Content,
			}
			raw, err = json.Marshal([]anthropicContentBlock{block})
			if err != nil {
				return nil, err
			}
			result = append(result, anthropicMessage{Role: "user", Content: raw})

		case "assistant":
			if len(m.ToolCalls) > 0 {
				// Assistant message with tool_use blocks — reconstruct structured content.
				var blocks []anthropicContentBlock
				if m.Content != "" {
					blocks = append(blocks, anthropicContentBlock{Type: "text", Text: m.Content})
				}
				for _, tc := range m.ToolCalls {
					input := make(map[string]any, len(tc.Params))
					for k, v := range tc.Params {
						input[k] = v
					}
					blocks = append(blocks, anthropicContentBlock{
						Type:  "tool_use",
						ID:    tc.ID,
						Name:  tc.Name,
						Input: input,
					})
				}
				raw, err = json.Marshal(blocks)
				if err != nil {
					return nil, err
				}
				result = append(result, anthropicMessage{Role: "assistant", Content: raw})
			} else {
				// Plain text assistant message.
				raw, err = json.Marshal(m.Content)
				if err != nil {
					return nil, err
				}
				result = append(result, anthropicMessage{Role: "assistant", Content: raw})
			}

		case "user":
			// Plain text content block.
			raw, err = json.Marshal(m.Content)
			if err != nil {
				return nil, err
			}
			result = append(result, anthropicMessage{Role: "user", Content: raw})

		default:
			// Skip unknown roles (e.g. "system" already extracted).
			continue
		}
	}
	return result, nil
}

// parseResponse converts an anthropicResponse into an llm.Response.
func parseResponse(apiResp *anthropicResponse) *Response {
	resp := &Response{
		StopReason:   apiResp.StopReason,
		PromptTok:    apiResp.Usage.InputTokens,
		OutputTok:    apiResp.Usage.OutputTokens,
		CacheCreated: apiResp.Usage.CacheCreationInputTokens,
		CacheHit:     apiResp.Usage.CacheReadInputTokens,
	}

	for _, block := range apiResp.Content {
		switch block.Type {
		case "thinking":
			// Extended thinking output — consumed for token accounting
			// but not included in Content (internal reasoning).
			continue

		case "text":
			if resp.Content != "" {
				resp.Content += "\n"
			}
			resp.Content += block.Text

		case "tool_use":
			params := make(map[string]string, len(block.Input))
			for k, v := range block.Input {
				params[k] = fmt.Sprintf("%v", v)
			}
			resp.ToolCalls = append(resp.ToolCalls, ToolCall{
				ID:     block.ID,
				Name:   block.Name,
				Params: params,
			})
		}
	}

	return resp
}
