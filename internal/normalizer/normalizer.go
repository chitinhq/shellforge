// Package normalizer converts raw tool calls into Canonical Action
// Representations (CARs). This is the bridge between the agent loop's
// free-form tool calls and the structured governance/orchestrator layer.
package normalizer

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/chitinhq/shellforge/internal/action"
	"github.com/chitinhq/chitin/canon"
)

// Normalize converts a raw tool call into a Canonical Action Representation.
func Normalize(runID string, sequence int, agent string, toolName string, params map[string]string) action.Proposal {
	actionType, risk, scope := classifyTool(toolName, params)
	target := extractTarget(toolName, params)

	// Convert params to map[string]any for the Proposal struct.
	anyParams := make(map[string]any, len(params))
	for k, v := range params {
		anyParams[k] = v
	}

	return action.Proposal{
		ID:       fmt.Sprintf("%s_%d", runID, sequence),
		RunID:    runID,
		Sequence: sequence,
		Agent:    agent,
		Type:     actionType,
		Target:   target,
		Params:   anyParams,
		Risk:     risk,
		Scope:    scope,
	}
}

// Fingerprint produces a deterministic hash of (type + target + sorted params)
// for anti-loop detection. Repeated identical proposals get the same fingerprint.
func Fingerprint(p action.Proposal) string {
	h := sha256.New()
	h.Write([]byte(string(p.Type)))
	h.Write([]byte("|"))
	h.Write([]byte(p.Target))
	h.Write([]byte("|"))

	// Sort param keys for deterministic hashing.
	keys := make([]string, 0, len(p.Params))
	for k := range p.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte("="))
		h.Write([]byte(fmt.Sprintf("%v", p.Params[k])))
		h.Write([]byte(";"))
	}

	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// classifyTool maps a tool name + params to (ActionType, RiskLevel, Scope).
func classifyTool(toolName string, params map[string]string) (action.ActionType, action.RiskLevel, action.Scope) {
	switch toolName {
	case "read_file":
		return action.FileRead, action.RiskReadOnly, action.ScopeFile

	case "write_file":
		return action.FileWrite, action.RiskMutating, action.ScopeFile

	case "run_shell":
		risk := classifyShellRisk(params["command"])
		return action.ShellExec, risk, action.ScopeSystem

	case "list_files":
		return action.FileRead, action.RiskReadOnly, action.ScopeDirectory

	case "search_files":
		return action.FileRead, action.RiskReadOnly, action.ScopeDirectory

	default:
		// Unknown tools default to mutating for safety.
		return action.ShellExec, action.RiskMutating, action.ScopeSystem
	}
}

// canonicalReadOnly are canonical tool names that only inspect state.
var canonicalReadOnly = map[string]bool{
	"read": true, "grep": true, "find": true, "ls": true,
	"echo": true, "wc": true, "diff": true, "sort": true, "uniq": true,
}

// canonicalDestructive are canonical tool+action combos with high blast radius.
var canonicalDestructive = map[string]bool{
	"rm":         true,
	"dd":         true,
	"git.push":   true,
	"git.reset":  true,
	"chmod":      true,
	"chown":      true,
	"kill":       true,
}

// classifyShellRisk inspects a shell command string to determine risk level.
// Uses canonical parsing for accurate classification.
func classifyShellRisk(command string) action.RiskLevel {
	cmd := canon.ParseOne(command)

	// Check destructive first.
	toolAction := cmd.Tool
	if cmd.Action != "" {
		toolAction = cmd.Tool + "." + cmd.Action
	}
	if canonicalDestructive[toolAction] || canonicalDestructive[cmd.Tool] {
		return action.RiskDestructive
	}

	// Check read-only.
	if canonicalReadOnly[cmd.Tool] {
		return action.RiskReadOnly
	}

	// Read-only git subcommands.
	if cmd.Tool == "git" {
		switch cmd.Action {
		case "status", "log", "diff", "show", "branch", "remote", "tag", "stash":
			return action.RiskReadOnly
		}
	}

	// Read-only go subcommands.
	if cmd.Tool == "go" {
		switch cmd.Action {
		case "test", "vet", "doc", "list", "version":
			return action.RiskReadOnly
		}
	}

	return action.RiskMutating
}

// ShellFingerprint returns the canonical digest of a shell command,
// suitable for deduplication and loop detection. Two semantically
// equivalent commands produce the same fingerprint.
func ShellFingerprint(command string) string {
	cmd := canon.ParseOne(command)
	return cmd.Digest
}

// extractTarget pulls the most relevant target path/identifier from params.
func extractTarget(toolName string, params map[string]string) string {
	switch toolName {
	case "read_file", "write_file":
		return params["path"]
	case "run_shell":
		return params["command"]
	case "list_files":
		return params["directory"]
	case "search_files":
		return params["directory"]
	default:
		return ""
	}
}
