package canon

import (
	"strings"
)

// rtkSubcommands are canonical tools that RTK has dedicated filters for.
// When routing through RTK, use `rtk <subcommand>` instead of `rtk sh -c`.
var rtkSubcommands = map[string]string{
	"git":     "git",
	"gh":      "gh",
	"grep":    "grep",
	"find":    "find",
	"ls":      "ls",
	"read":    "read",
	"docker":  "docker",
	"kubectl": "kubectl",
	"pnpm":    "pnpm",
	"npm":     "npm",
	"cargo":   "cargo",
	"go":      "go",
	"curl":    "curl",
	"wget":    "wget",
	"diff":    "diff",
	"tsc":     "tsc",
}

// RTKCommand rewrites a raw shell command into the optimal RTK invocation.
// Returns the rewritten args for exec.Command("rtk", args...) and whether
// a specific filter was found. If no filter matches, returns nil, false
// and the caller should use "rtk sh -c <command>" as fallback.
func RTKCommand(raw string) (args []string, specific bool) {
	pipeline := Parse(raw)
	if len(pipeline.Segments) != 1 {
		// Chains and pipes: can't use a single RTK subcommand.
		return nil, false
	}

	cmd := pipeline.Segments[0].Command

	sub, ok := rtkSubcommands[cmd.Tool]
	if !ok {
		return nil, false
	}

	// Reconstruct: rtk <sub> [action] [original flags and args]
	// We pass the raw trailing args (after the tool name) to preserve user intent.
	rawTrimmed := strings.TrimSpace(raw)
	tokens := tokenize(rawTrimmed)
	if len(tokens) == 0 {
		return nil, false
	}

	// tokens[0] is the raw command name (e.g., "git", "cat", "rg")
	// For tools like cat/head/tail that map to "read", use rtk read <file>
	if cmd.Tool == "read" {
		args = []string{sub}
		args = append(args, tokens[1:]...) // pass remaining args as-is
		return args, true
	}

	// For tools where raw name == canonical name (git, docker, etc.),
	// pass everything after the tool name.
	args = []string{sub}
	args = append(args, tokens[1:]...)
	return args, true
}

// maxCompressedLines is the default line limit for compressed output.
const maxCompressedLines = 50

// CompressOutput applies tool-aware compression to command output.
// This is used when RTK isn't available or as a post-filter.
func CompressOutput(cmd Command, output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) <= maxCompressedLines {
		return output
	}

	switch cmd.Tool {
	case "git":
		return compressGit(cmd.Action, lines)
	case "grep":
		return compressGrep(lines)
	case "read":
		return compressRead(lines)
	case "ls":
		return compressLS(lines)
	default:
		return compressGeneric(lines)
	}
}

func compressGit(action string, lines []string) string {
	switch action {
	case "log":
		// Keep first 30 commits.
		if len(lines) > 30 {
			return strings.Join(lines[:30], "\n") + "\n... (" + itoa(len(lines)-30) + " more lines)"
		}
	case "diff", "show":
		// Keep file headers and first few lines of each hunk.
		return compressDiff(lines)
	case "status":
		// Status is usually short; keep first 20 lines.
		if len(lines) > 20 {
			return strings.Join(lines[:20], "\n") + "\n... (" + itoa(len(lines)-20) + " more files)"
		}
	}
	return compressGeneric(lines)
}

func compressDiff(lines []string) string {
	var out []string
	hunksKept := 0
	inHunk := false
	hunkLines := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			out = append(out, line)
			inHunk = false
			continue
		}
		if strings.HasPrefix(line, "@@") {
			out = append(out, line)
			inHunk = true
			hunkLines = 0
			hunksKept++
			continue
		}
		if inHunk {
			hunkLines++
			if hunkLines <= 10 {
				out = append(out, line)
			} else if hunkLines == 11 {
				out = append(out, "  ... (hunk truncated)")
			}
		}
	}

	if hunksKept > 0 {
		return strings.Join(out, "\n")
	}
	return compressGeneric(lines)
}

func compressGrep(lines []string) string {
	// Keep first 5 matches per file, max 30 files.
	var out []string
	currentFile := ""
	matchesInFile := 0
	files := 0

	for _, line := range lines {
		// Detect file header (grep output: "file:line:match")
		if idx := strings.Index(line, ":"); idx > 0 {
			file := line[:idx]
			if file != currentFile {
				currentFile = file
				matchesInFile = 0
				files++
				if files > 30 {
					out = append(out, "... (more files omitted)")
					break
				}
			}
			matchesInFile++
			if matchesInFile <= 5 {
				out = append(out, line)
			} else if matchesInFile == 6 {
				out = append(out, "  ... (more matches in "+currentFile+")")
			}
		} else {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func compressRead(lines []string) string {
	// For file reads, keep first and last 20 lines.
	if len(lines) <= 40 {
		return strings.Join(lines, "\n")
	}
	head := lines[:20]
	tail := lines[len(lines)-20:]
	omitted := len(lines) - 40
	return strings.Join(head, "\n") + "\n... (" + itoa(omitted) + " lines omitted)\n" + strings.Join(tail, "\n")
}

func compressLS(lines []string) string {
	// Keep first 30 entries.
	if len(lines) > 30 {
		return strings.Join(lines[:30], "\n") + "\n... (" + itoa(len(lines)-30) + " more entries)"
	}
	return strings.Join(lines, "\n")
}

func compressGeneric(lines []string) string {
	if len(lines) <= maxCompressedLines {
		return strings.Join(lines, "\n")
	}
	head := lines[:maxCompressedLines/2]
	tail := lines[len(lines)-maxCompressedLines/2:]
	omitted := len(lines) - maxCompressedLines
	return strings.Join(head, "\n") + "\n... (" + itoa(omitted) + " lines omitted)\n" + strings.Join(tail, "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
