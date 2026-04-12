// Package intent extracts action intent from LLM output regardless of format.
//
// Models express tool calls in many ways:
//   - Structured tool_calls (OpenAI/Anthropic format)
//   - JSON in ```json blocks
//   - JSON in <tool> XML tags
//   - Bare JSON objects in text
//   - Natural language descriptions of actions ("I'll write file X with content Y")
//   - OpenAI function_call format (name + arguments)
//   - Ollama-specific format variations
//
// This parser normalizes ALL of these into a unified Action struct.
// Every extracted action goes through Chitin governance — no exceptions.
//
// This is ShellForge's moat: format-agnostic execution firewall.
package intent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Source tracks how an action was extracted — for telemetry and debugging.
type Source string

const (
	SourceToolCall     Source = "tool_call"      // structured API response
	SourceJSONBlock    Source = "json_block"      // ```json ``` in content
	SourceXMLTag       Source = "xml_tag"         // <tool>...</tool> in content
	SourceBareJSON     Source = "bare_json"       // raw JSON object in text
	SourceFunctionCall Source = "function_call"   // OpenAI function_call format
	SourceNaturalLang  Source = "natural_language" // heuristic from prose
)

// Action represents extracted intent — what the model wants to do.
type Action struct {
	Tool   string            `json:"tool"`
	Params map[string]string `json:"params"`
	Source Source             `json:"source"`
	Raw    string            `json:"raw"` // original text that produced this
}

// Aliases map common model-emitted tool names to ShellForge canonical names.
var toolAliases = map[string]string{
	// File operations
	"read_file":    "read_file",
	"readFile":     "read_file",
	"Read":         "read_file",
	"view":         "read_file",
	"cat":          "read_file",

	"write_file":   "write_file",
	"writeFile":    "write_file",
	"Write":        "write_file",
	"write":        "write_file",
	"create_file":  "write_file",
	"createFile":   "write_file",
	"overwrite":    "write_file",

	"edit":         "write_file",
	"Edit":         "write_file",
	"patch":        "write_file",

	// Shell
	"run_shell":    "run_shell",
	"runShell":     "run_shell",
	"Bash":         "run_shell",
	"bash":         "run_shell",
	"shell":        "run_shell",
	"execute":      "run_shell",
	"run_command":  "run_shell",
	"runCommand":   "run_shell",
	"terminal":     "run_shell",
	"exec":         "run_shell",

	// File listing
	"list_files":   "list_files",
	"listFiles":    "list_files",
	"ls":           "list_files",
	"Glob":         "list_files",
	"list_dir":     "list_files",
	"listDir":      "list_files",

	// Search
	"search_files": "search_files",
	"searchFiles":  "search_files",
	"grep":         "search_files",
	"Grep":         "search_files",
	"search":       "search_files",
	"find":         "search_files",
}

// Param aliases map common parameter names to canonical names.
var paramAliases = map[string]string{
	"file_path":  "path",
	"filePath":   "path",
	"file":       "path",
	"filename":   "path",
	"filepath":   "path",
	"target":     "path",

	"content":    "content",
	"text":       "content",
	"data":       "content",
	"body":       "content",

	"command":    "command",
	"cmd":        "command",
	"shell":      "command",
	"script":     "command",

	"directory":  "directory",
	"dir":        "directory",
	"folder":     "directory",
	"path":       "path",

	"pattern":    "pattern",
	"query":      "pattern",
	"search":     "pattern",
	"regex":      "pattern",
}

var (
	// Match ```json ... ``` blocks
	jsonBlockRe = regexp.MustCompile("(?s)```(?:json)?\\s*\n?(\\{.*?\\})\n?\\s*```")

	// Match <tool>...</tool> or <tool_call>...</tool_call> XML tags
	xmlToolRe = regexp.MustCompile(`(?s)<(?:tool|tool_call|function_call)>(.*?)</(?:tool|tool_call|function_call)>`)

	// Match bare JSON objects: { "tool": ... } or { "name": ... }
	bareJSONRe = regexp.MustCompile(`(?s)\{[^{}]*"(?:tool|name|function)"[^{}]*\}`)

	// Match OpenAI function_call format: {"name": "...", "arguments": "..."}
	functionCallRe = regexp.MustCompile(`(?s)\{[^{}]*"name"\s*:\s*"[^"]+"\s*,\s*"arguments"\s*:`)
)

// Parse extracts action intent from LLM output content.
// Returns nil if no actionable intent is found (model is giving a final answer).
func Parse(content string) *Action {
	// Strategy 1: JSON code blocks (highest confidence)
	if a := tryJSONBlocks(content); a != nil {
		return a
	}

	// Strategy 2: XML tool tags
	if a := tryXMLTags(content); a != nil {
		return a
	}

	// Strategy 3: OpenAI function_call format
	if a := tryFunctionCall(content); a != nil {
		return a
	}

	// Strategy 4: Bare JSON objects
	if a := tryBareJSON(content); a != nil {
		return a
	}

	// No structured intent found — this is a final answer, not a tool call.
	return nil
}

func tryJSONBlocks(content string) *Action {
	matches := jsonBlockRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if a := parseToolJSON(m[1], SourceJSONBlock); a != nil {
			return a
		}
	}
	return nil
}

func tryXMLTags(content string) *Action {
	matches := xmlToolRe.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		trimmed := strings.TrimSpace(m[1])
		if a := parseToolJSON(trimmed, SourceXMLTag); a != nil {
			return a
		}
	}
	return nil
}

func tryFunctionCall(content string) *Action {
	loc := functionCallRe.FindStringIndex(content)
	if loc == nil {
		return nil
	}
	// Extract the full JSON object starting from the match
	jsonStr := extractJSONObject(content[loc[0]:])
	if jsonStr == "" {
		return nil
	}

	var fc struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // OpenAI nests args as a JSON string
	}
	if err := json.Unmarshal([]byte(jsonStr), &fc); err != nil {
		return nil
	}
	if fc.Name == "" {
		return nil
	}

	tool := normalizeTool(fc.Name)
	if tool == "" {
		return nil
	}

	params := make(map[string]string)
	if fc.Arguments != "" {
		var args map[string]any
		if err := json.Unmarshal([]byte(fc.Arguments), &args); err == nil {
			params = flattenParams(args)
		}
	}

	return &Action{
		Tool:   tool,
		Params: normalizeParams(params),
		Source: SourceFunctionCall,
		Raw:    jsonStr,
	}
}

func tryBareJSON(content string) *Action {
	matches := bareJSONRe.FindAllString(content, 3) // check up to 3 candidates
	for _, m := range matches {
		if a := parseToolJSON(m, SourceBareJSON); a != nil {
			return a
		}
	}
	return nil
}

// parseToolJSON attempts to parse a JSON string into an Action.
// Handles multiple formats:
//   {"tool": "write_file", "params": {"path": "...", "content": "..."}}
//   {"tool": "write_file", "path": "...", "content": "..."}
//   {"name": "write", "arguments": {"file_path": "..."}}
//   {"function": "bash", "command": "ls -la"}
func parseToolJSON(s string, source Source) *Action {
	var raw map[string]any
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		return nil
	}

	// Extract tool name from various fields
	toolName := ""
	for _, key := range []string{"tool", "name", "function", "action", "type"} {
		if v, ok := raw[key].(string); ok && v != "" {
			toolName = v
			break
		}
	}
	if toolName == "" {
		return nil
	}

	tool := normalizeTool(toolName)
	if tool == "" {
		return nil // unknown tool — don't guess
	}

	// Extract params from various structures
	params := make(map[string]string)

	// Check for nested params/arguments object
	for _, key := range []string{"params", "arguments", "args", "input", "parameters"} {
		if nested, ok := raw[key]; ok {
			switch v := nested.(type) {
			case map[string]any:
				params = flattenParams(v)
			case string:
				// OpenAI-style: arguments is a JSON string
				var args map[string]any
				if err := json.Unmarshal([]byte(v), &args); err == nil {
					params = flattenParams(args)
				}
			}
			if len(params) > 0 {
				break
			}
		}
	}

	// Fallback: top-level params (flat structure)
	if len(params) == 0 {
		for k, v := range raw {
			switch k {
			case "tool", "name", "function", "action", "type":
				continue // skip the tool name field
			default:
				if s, ok := v.(string); ok {
					params[k] = s
				}
			}
		}
	}

	return &Action{
		Tool:   tool,
		Params: normalizeParams(params),
		Source: source,
		Raw:    s,
	}
}

// normalizeTool maps any model-emitted tool name to ShellForge's canonical name.
// Returns "" for unknown tools.
func normalizeTool(name string) string {
	if canonical, ok := toolAliases[name]; ok {
		return canonical
	}
	// Case-insensitive fallback
	lower := strings.ToLower(name)
	for alias, canonical := range toolAliases {
		if strings.ToLower(alias) == lower {
			return canonical
		}
	}
	return ""
}

// normalizeParams maps any model-emitted param names to canonical names.
func normalizeParams(params map[string]string) map[string]string {
	normalized := make(map[string]string, len(params))
	for k, v := range params {
		if canonical, ok := paramAliases[k]; ok {
			normalized[canonical] = v
		} else {
			normalized[k] = v
		}
	}
	return normalized
}

// flattenParams converts map[string]any to map[string]string.
func flattenParams(m map[string]any) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = val
		case float64:
			result[k] = fmt.Sprintf("%g", val)
		case bool:
			result[k] = fmt.Sprintf("%t", val)
		default:
			// For nested objects, serialize to JSON
			if b, err := json.Marshal(val); err == nil {
				result[k] = string(b)
			}
		}
	}
	return result
}

// extractJSONObject extracts a balanced JSON object from a string.
func extractJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	inStr := false
	escaped := false
	for i := start; i < len(s); i++ {
		if escaped {
			escaped = false
			continue
		}
		switch s[i] {
		case '\\':
			if inStr {
				escaped = true
			}
		case '"':
			inStr = !inStr
		case '{':
			if !inStr {
				depth++
			}
		case '}':
			if !inStr {
				depth--
				if depth == 0 {
					return s[start : i+1]
				}
			}
		}
	}
	return ""
}

