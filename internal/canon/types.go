// Package canon provides shell command canonicalization — parsing raw shell
// strings into structured, normalized forms suitable for deduplication,
// caching, and telemetry analysis.
//
// Two commands that do the same thing should produce the same canonical form
// and the same digest. For example: "cat foo.txt", "head foo.txt", and
// "tail foo.txt" all canonicalize to Tool="read", Args=["foo.txt"] with
// identical digests.
package canon

// Command is the canonical representation of a single shell command.
type Command struct {
	Tool   string            `json:"tool"`           // Canonical tool: "git", "read", "grep", "docker", etc.
	Action string            `json:"action,omitempty"` // Subcommand: "status", "log", "diff", etc.
	Flags  map[string]string `json:"flags,omitempty"`  // Normalized flags (long form, sorted by key).
	Args   []string          `json:"args,omitempty"`   // Positional arguments (paths, patterns, etc.).
	Raw    string            `json:"raw"`              // Original command string.
	Digest string            `json:"digest"`           // SHA256 of canonical form (first 16 hex chars).
}

// Pipeline represents a sequence of commands connected by pipes or chain operators.
type Pipeline struct {
	Segments []Segment `json:"segments"`
}

// Segment is one command in a pipeline, preceded by an operator (except the first).
type Segment struct {
	Op      ChainOp `json:"op,omitempty"` // "", "|", "&&", "||", ";"
	Command Command `json:"command"`
}

// ChainOp is the operator connecting two commands.
type ChainOp string

const (
	OpNone  ChainOp = ""
	OpPipe  ChainOp = "|"
	OpAnd   ChainOp = "&&"
	OpOr    ChainOp = "||"
	OpSeq   ChainOp = ";"
)

// toolAliases maps raw command names to canonical tool names.
// Commands that read files all map to "read", search tools map to "grep", etc.
var toolAliases = map[string]string{
	// File readers → "read"
	"cat":  "read",
	"head": "read",
	"tail": "read",
	"less": "read",
	"more": "read",
	"bat":  "read",

	// Search → "grep"
	"grep":    "grep",
	"rg":      "grep",
	"ripgrep": "grep",
	"ag":      "grep",
	"ack":     "grep",

	// Find → "find"
	"find": "find",
	"fd":   "find",

	// List → "ls"
	"ls":   "ls",
	"exa":  "ls",
	"eza":  "ls",
	"tree": "ls",

	// Git stays "git"
	"git": "git",

	// Docker stays "docker"
	"docker": "docker",

	// Kubernetes → "kubectl"
	"kubectl": "kubectl",
	"k":       "kubectl",

	// Package managers
	"pnpm": "pnpm",
	"npm":  "npm",
	"npx":  "npx",
	"yarn": "yarn",
	"pip":  "pip",
	"uv":   "uv",

	// Build tools
	"cargo":  "cargo",
	"go":     "go",
	"make":   "make",
	"tsc":    "tsc",
	"python": "python",
	"python3": "python",
	"node":   "node",

	// GitHub CLI
	"gh": "gh",

	// Curl/wget
	"curl": "curl",
	"wget": "wget",

	// Misc
	"cd":    "cd",
	"echo":  "echo",
	"rm":    "rm",
	"cp":    "cp",
	"mv":    "mv",
	"mkdir": "mkdir",
	"chmod": "chmod",
	"chown": "chown",
	"kill":  "kill",
	"sed":   "sed",
	"awk":   "awk",
	"sort":  "sort",
	"uniq":  "uniq",
	"wc":    "wc",
	"diff":  "diff",
	"patch": "patch",
	"dd":    "dd",
}

// flagAliases maps short flags to long forms for specific tools.
// Key format: "tool:short" → "long" (without leading dashes).
var flagAliases = map[string]string{
	// grep
	"grep:-r": "recursive",
	"grep:-i": "ignore-case",
	"grep:-n": "line-number",
	"grep:-l": "files-with-matches",
	"grep:-c": "count",
	"grep:-v": "invert-match",
	"grep:-w": "word-regexp",
	"grep:-e": "regexp",
	"grep:-A": "after-context",
	"grep:-B": "before-context",
	"grep:-C": "context",

	// git log
	"git.log:-n":       "max-count",
	"git.log:--oneline": "format=oneline",

	// read (head/tail/cat)
	"read:-n": "lines",

	// ls
	"ls:-l": "long",
	"ls:-a": "all",
	"ls:-h": "human-readable",
	"ls:-R": "recursive",

	// find
	"find:-name":  "name",
	"find:-type":  "type",
	"find:-maxdepth": "maxdepth",

	// docker
	"docker.ps:-a": "all",
	"docker.ps:-q": "quiet",

	// curl
	"curl:-s": "silent",
	"curl:-o": "output",
	"curl:-X": "request",
	"curl:-H": "header",
	"curl:-d": "data",
	"curl:-L": "location",
}

// sensitivePatterns are environment variable names whose values should be masked.
var sensitivePatterns = []string{
	"API_KEY", "SECRET", "TOKEN", "PASSWORD", "CREDENTIAL",
	"PRIVATE_KEY", "AUTH",
}
