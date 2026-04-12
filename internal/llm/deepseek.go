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
	defaultDeepSeekModel   = "deepseek-chat"
	defaultDeepSeekBaseURL = "https://api.deepseek.com"
)

// DeepSeekProvider calls the DeepSeek API (OpenAI-compatible) via stdlib HTTP.
type DeepSeekProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewDeepSeekProvider creates a DeepSeekProvider.
// Pass empty strings to fall back to DEEPSEEK_API_KEY and DEEPSEEK_MODEL env vars.
func NewDeepSeekProvider(apiKey, model string) *DeepSeekProvider {
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	if model == "" {
		model = envOr("DEEPSEEK_MODEL", defaultDeepSeekModel)
	}
	return &DeepSeekProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: defaultDeepSeekBaseURL,
		client:  &http.Client{Timeout: 5 * time.Minute},
	}
}

// Name returns the provider identifier.
func (d *DeepSeekProvider) Name() string {
	return "deepseek"
}

// OpenAI-compatible wire types

type oaiMessage struct {
	Role       string        `json:"role"`
	Content    string        `json:"content,omitempty"`
	ToolCalls  []oaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string        `json:"tool_call_id,omitempty"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiToolFunction `json:"function"`
}

type oaiToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiToolDef struct {
	Type     string         `json:"type"`
	Function oaiFunctionDef `json:"function"`
}

type oaiFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type oaiRequest struct {
	Model    string       `json:"model"`
	Messages []oaiMessage `json:"messages"`
	Tools    []oaiToolDef `json:"tools,omitempty"`
}

type oaiResponse struct {
	Choices []struct {
		Message      oaiMessage `json:"message"`
		FinishReason string     `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type oaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// parseToolArgs converts JSON arguments string to map[string]string.
// Handles the common case where DeepSeek returns non-string values (numbers,
// booleans) or content with special characters that break map[string]string
// unmarshaling. Falls back to map[string]any and stringifies values.
func parseToolArgs(raw string) map[string]string {
	// Try direct string map first (cheapest path).
	var direct map[string]string
	if err := json.Unmarshal([]byte(raw), &direct); err == nil && direct != nil {
		return direct
	}

	// Fall back to any-typed values and stringify.
	var generic map[string]any
	if err := json.Unmarshal([]byte(raw), &generic); err == nil && generic != nil {
		result := make(map[string]string, len(generic))
		for k, v := range generic {
			switch val := v.(type) {
			case string:
				result[k] = val
			default:
				result[k] = fmt.Sprintf("%v", val)
			}
		}
		return result
	}

	return make(map[string]string)
}

// Chat sends messages to the DeepSeek API and returns the response.
func (d *DeepSeekProvider) Chat(messages []Message, tools []ToolDef) (*Response, error) {
	apiMsgs := make([]oaiMessage, 0, len(messages))
	for _, m := range messages {
		switch m.Role {
		case "tool_result":
			apiMsgs = append(apiMsgs, oaiMessage{
				Role:       "tool",
				Content:    m.Content,
				ToolCallID: m.ToolCallID,
			})
		case "assistant":
			msg := oaiMessage{Role: "assistant", Content: m.Content}
			for _, tc := range m.ToolCalls {
				params, _ := json.Marshal(tc.Params)
				msg.ToolCalls = append(msg.ToolCalls, oaiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: oaiToolFunction{
						Name:      tc.Name,
						Arguments: string(params),
					},
				})
			}
			apiMsgs = append(apiMsgs, msg)
		default:
			apiMsgs = append(apiMsgs, oaiMessage{Role: m.Role, Content: m.Content})
		}
	}

	apiTools := make([]oaiToolDef, len(tools))
	for i, t := range tools {
		schema := t.Parameters
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		apiTools[i] = oaiToolDef{
			Type: "function",
			Function: oaiFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schema,
			},
		}
	}

	reqBody := oaiRequest{
		Model:    d.model,
		Messages: apiMsgs,
	}
	if len(apiTools) > 0 {
		reqBody.Tools = apiTools
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("deepseek: marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, d.baseURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("deepseek: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+d.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deepseek: http request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("deepseek: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr oaiErrorResponse
		if jsonErr := json.Unmarshal(respBytes, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("deepseek: api error %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("deepseek: http %d: %s", resp.StatusCode, string(respBytes))
	}

	var apiResp oaiResponse
	if err := json.Unmarshal(respBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("deepseek: unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("deepseek: empty response (no choices)")
	}

	choice := apiResp.Choices[0]
	result := &Response{
		Content:    choice.Message.Content,
		StopReason: choice.FinishReason,
		PromptTok:  apiResp.Usage.PromptTokens,
		OutputTok:  apiResp.Usage.CompletionTokens,
	}

	for _, tc := range choice.Message.ToolCalls {
		params := parseToolArgs(tc.Function.Arguments)
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:     tc.ID,
			Name:   tc.Function.Name,
			Params: params,
		})
	}

	return result, nil
}
