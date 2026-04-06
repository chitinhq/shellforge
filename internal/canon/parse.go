package canon

import (
	"strings"
)

// Parse takes a raw shell command string and returns a Pipeline of canonical commands.
// It handles chains (&&, ||, ;) and pipes (|), normalizing each segment.
func Parse(raw string) Pipeline {
	segments := splitChain(raw)
	pipeline := Pipeline{Segments: make([]Segment, 0, len(segments))}

	for _, seg := range segments {
		cmd := parseOne(seg.text)
		cmd.Raw = strings.TrimSpace(seg.text)
		cmd.Digest = Digest(cmd)
		pipeline.Segments = append(pipeline.Segments, Segment{
			Op:      seg.op,
			Command: cmd,
		})
	}
	return pipeline
}

// ParseOne parses a single command (no chains/pipes) into canonical form.
func ParseOne(raw string) Command {
	cmd := parseOne(raw)
	cmd.Raw = strings.TrimSpace(raw)
	cmd.Digest = Digest(cmd)
	return cmd
}

// chainSegment is an intermediate parse result: operator + raw text.
type chainSegment struct {
	op   ChainOp
	text string
}

// splitChain splits a command string on &&, ||, ;, and | operators.
// Respects single and double quotes (does not split inside them).
func splitChain(raw string) []chainSegment {
	var segments []chainSegment
	var current strings.Builder
	currentOp := OpNone

	i := 0
	inSingle := false
	inDouble := false

	for i < len(raw) {
		ch := raw[i]

		// Track quote state.
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			current.WriteByte(ch)
			i++
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			current.WriteByte(ch)
			i++
			continue
		}

		// Skip escaped characters.
		if ch == '\\' && i+1 < len(raw) {
			current.WriteByte(ch)
			current.WriteByte(raw[i+1])
			i += 2
			continue
		}

		// Only split outside quotes.
		if !inSingle && !inDouble {
			// Check && first (before single &).
			if ch == '&' && i+1 < len(raw) && raw[i+1] == '&' {
				text := current.String()
				if strings.TrimSpace(text) != "" {
					segments = append(segments, chainSegment{op: currentOp, text: text})
				}
				current.Reset()
				currentOp = OpAnd
				i += 2
				continue
			}
			// Check ||.
			if ch == '|' && i+1 < len(raw) && raw[i+1] == '|' {
				text := current.String()
				if strings.TrimSpace(text) != "" {
					segments = append(segments, chainSegment{op: currentOp, text: text})
				}
				current.Reset()
				currentOp = OpOr
				i += 2
				continue
			}
			// Check single | (pipe).
			if ch == '|' {
				text := current.String()
				if strings.TrimSpace(text) != "" {
					segments = append(segments, chainSegment{op: currentOp, text: text})
				}
				current.Reset()
				currentOp = OpPipe
				i++
				continue
			}
			// Check ;.
			if ch == ';' {
				text := current.String()
				if strings.TrimSpace(text) != "" {
					segments = append(segments, chainSegment{op: currentOp, text: text})
				}
				current.Reset()
				currentOp = OpSeq
				i++
				continue
			}
		}

		current.WriteByte(ch)
		i++
	}

	// Final segment.
	text := current.String()
	if strings.TrimSpace(text) != "" {
		segments = append(segments, chainSegment{op: currentOp, text: text})
	}

	return segments
}

// parseOne parses a single command (no operators) into a Command.
func parseOne(raw string) Command {
	tokens := tokenize(strings.TrimSpace(raw))
	if len(tokens) == 0 {
		return Command{Tool: "unknown"}
	}

	// Handle env var prefix: KEY=val KEY2=val2 command args...
	cmdIdx := 0
	for cmdIdx < len(tokens) {
		if strings.Contains(tokens[cmdIdx], "=") && !strings.HasPrefix(tokens[cmdIdx], "-") {
			cmdIdx++
		} else {
			break
		}
	}
	if cmdIdx >= len(tokens) {
		return Command{Tool: "env", Args: tokens}
	}

	cmdName := tokens[cmdIdx]
	rest := tokens[cmdIdx+1:]

	// Resolve tool alias.
	tool := cmdName
	if alias, ok := toolAliases[cmdName]; ok {
		tool = alias
	}

	// Extract action (subcommand) for multi-level tools.
	action := ""
	argStart := 0
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		switch tool {
		case "git", "docker", "kubectl", "gh", "cargo", "go", "pnpm", "npm", "uv":
			action = rest[0]
			argStart = 1
		}
	}

	// Parse flags and positional args.
	flags := make(map[string]string)
	var args []string

	remaining := rest[argStart:]
	for i := 0; i < len(remaining); i++ {
		tok := remaining[i]

		if strings.HasPrefix(tok, "-") {
			// Expand combined short flags: -rn → [-r, -n]
			expanded := expandShortFlags(tok)
			for j, expFlag := range expanded {
				flagName, flagVal := parseFlag(expFlag)

				// If flag has no inline value, check if next token is the value.
				// Only the last expanded flag can consume the next token.
				if flagVal == "" && j == len(expanded)-1 && i+1 < len(remaining) && !strings.HasPrefix(remaining[i+1], "-") {
					if flagTakesValue(tool, action, flagName) {
						flagVal = remaining[i+1]
						i++
					}
				}

				canonical := normalizeFlag(tool, action, flagName)
				if flagVal != "" {
					flagVal = maskSensitive(flagVal)
				}
				flags[canonical] = flagVal
			}
		} else {
			args = append(args, maskSensitive(tok))
		}
	}

	// Apply tool-specific normalizations.
	normalizeToolSpecific(tool, cmdName, action, flags, &args)

	return Command{
		Tool:   tool,
		Action: action,
		Flags:  flags,
		Args:   args,
	}
}

// tokenize splits a command string into tokens, respecting quotes.
func tokenize(s string) []string {
	var tokens []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(s); i++ {
		ch := s[i]

		if ch == '\\' && i+1 < len(s) && !inSingle {
			current.WriteByte(s[i+1])
			i++
			continue
		}

		if ch == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}

		if (ch == ' ' || ch == '\t') && !inSingle && !inDouble {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteByte(ch)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// parseFlag splits a flag token into name and inline value.
// Examples: "--count=5" → ("count", "5"), "-n" → ("n", ""), "--verbose" → ("verbose", "")
func parseFlag(tok string) (string, string) {
	// Strip leading dashes.
	stripped := strings.TrimLeft(tok, "-")
	if idx := strings.Index(stripped, "="); idx >= 0 {
		return stripped[:idx], stripped[idx+1:]
	}
	return stripped, ""
}

// expandShortFlags splits combined short flags into individual flags.
// "-rn" → ["-r", "-n"], "--verbose" → ["--verbose"], "-n" → ["-n"], "-C5" → ["-C5"]
func expandShortFlags(tok string) []string {
	// Long flags (--) are never combined.
	if strings.HasPrefix(tok, "--") {
		return []string{tok}
	}
	// Must be single dash.
	if !strings.HasPrefix(tok, "-") || len(tok) < 2 {
		return []string{tok}
	}

	body := tok[1:] // strip leading -

	// If it contains =, it's a valued flag: don't expand.
	if strings.Contains(body, "=") {
		return []string{tok}
	}

	// If it's a single character, nothing to expand.
	if len(body) == 1 {
		return []string{tok}
	}

	// If second char is a digit, it's a valued flag like -C5, -n20.
	if body[1] >= '0' && body[1] <= '9' {
		return []string{tok}
	}

	// Expand: -rn → [-r, -n]
	result := make([]string, len(body))
	for i, ch := range body {
		result[i] = "-" + string(ch)
	}
	return result
}
