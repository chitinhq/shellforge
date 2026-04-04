package tools

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"regexp"
"strings"

"github.com/AgentGuardHQ/shellforge/internal/governance"
"github.com/AgentGuardHQ/shellforge/internal/logger"
)

const MaxOutput = 2000

type Definition struct {
Name        string
Description string
Params      []Param
}

type Param struct {
Name     string
Type     string
Desc     string
Required bool
}

type Result struct {
Success bool
Output  string
Error   string
}

var Definitions = []Definition{
{
Name:        "read_file",
Description: "Read a file's contents",
Params:      []Param{{Name: "path", Type: "string", Desc: "File path to read", Required: true}},
},
{
Name:        "write_file",
Description: "Write content to a file (creates dirs if needed)",
Params: []Param{
{Name: "path", Type: "string", Desc: "File path to write", Required: true},
{Name: "content", Type: "string", Desc: "Content to write", Required: true},
},
},
{
Name:        "run_shell",
Description: "Run a shell command, return stdout/stderr",
Params:      []Param{{Name: "command", Type: "string", Desc: "Shell command to execute", Required: true}},
},
{
Name:        "list_files",
Description: "List files in a directory",
Params: []Param{
{Name: "directory", Type: "string", Desc: "Directory to list", Required: true},
{Name: "extension", Type: "string", Desc: "Filter by extension (e.g. .ts)", Required: false},
},
},
{
Name:        "search_files",
Description: "Search file contents for a pattern",
Params: []Param{
{Name: "pattern", Type: "string", Desc: "Text to search for", Required: true},
{Name: "directory", Type: "string", Desc: "Directory to search", Required: true},
},
},
{
Name:        "edit_file",
Description: "Apply a targeted edit to a file. Finds old_text and replaces it with new_text. Fails if old_text is not found or appears multiple times.",
Params: []Param{
{Name: "path", Type: "string", Desc: "File path to edit", Required: true},
{Name: "old_text", Type: "string", Desc: "Exact text to find and replace", Required: true},
{Name: "new_text", Type: "string", Desc: "Replacement text", Required: true},
},
},
{
Name:        "glob",
Description: "Find files matching a glob pattern",
Params: []Param{
{Name: "pattern", Type: "string", Desc: "Glob pattern (e.g. **/*.go, *.ts)", Required: true},
{Name: "directory", Type: "string", Desc: "Directory to search in", Required: false},
},
},
{
Name:        "grep",
Description: "Search file contents for a regex pattern, returning matching lines with file:line format",
Params: []Param{
{Name: "pattern", Type: "string", Desc: "Regex pattern to search for", Required: true},
{Name: "directory", Type: "string", Desc: "Directory to search in", Required: false},
{Name: "file_type", Type: "string", Desc: "File extension filter (e.g. go, ts, py)", Required: false},
},
},
}

// ExecuteDirect runs a tool implementation without governance evaluation.
// Use this when governance has already been checked by the caller (e.g., the agent loop).
func ExecuteDirect(tool string, params map[string]string, timeoutSec int) Result {
	impl, ok := impls[tool]
	if !ok {
		return Result{Success: false, Output: "Unknown tool: " + tool, Error: "unknown_tool"}
	}
	return impl(params, timeoutSec)
}

// Execute runs a tool call through governance, then executes if allowed.
// Execute evaluates the tool call against governance policy and, if allowed, runs it.
// This is the fully governed path; use ExecuteDirect when governance is already checked.
func Execute(engine *governance.Engine, agent, tool string, params map[string]string) Result {
decision := engine.Evaluate(tool, params)
logger.Governance(agent, tool, params, decision.Allowed, decision.PolicyName, decision.Reason)

if !decision.Allowed {
return Result{
Success: false,
Output:  fmt.Sprintf("DENIED by policy %q: %s", decision.PolicyName, decision.Reason),
Error:   decision.Reason,
}
}

impl, ok := impls[tool]
if !ok {
return Result{Success: false, Output: "Unknown tool: " + tool, Error: "unknown_tool"}
}

result := impl(params, engine.GetTimeout())
logger.ToolResult(agent, tool, result.Success, result.Output)
return result
}

type implFunc func(params map[string]string, timeoutSec int) Result

var impls = map[string]implFunc{
"read_file":    readFile,
"write_file":   writeFile,
"run_shell":    runShell,
"list_files":   listFiles,
"search_files": searchFiles,
"edit_file":    editFile,
"glob":         globFiles,
"grep":         grepFiles,
}

func readFile(params map[string]string, _ int) Result {
path := params["path"]
info, err := os.Stat(path)
if err != nil {
return Result{Success: false, Output: "File not found: " + path, Error: "not_found"}
}
if info.Size() > 100_000 {
return Result{Success: false, Output: fmt.Sprintf("File too large (%dKB > 100KB)", info.Size()/1024), Error: "too_large"}
}
data, err := os.ReadFile(path)
if err != nil {
return Result{Success: false, Output: err.Error(), Error: "read_error"}
}
content := string(data)
if len(content) > MaxOutput {
return Result{Success: true, Output: content[:MaxOutput] + "\n... (truncated)"}
}
return Result{Success: true, Output: content}
}

func writeFile(params map[string]string, _ int) Result {
path := params["path"]
content := params["content"]
dir := filepath.Dir(path)
if err := os.MkdirAll(dir, 0o755); err != nil {
return Result{Success: false, Output: err.Error(), Error: "mkdir_error"}
}
if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
return Result{Success: false, Output: err.Error(), Error: "write_error"}
}
return Result{Success: true, Output: fmt.Sprintf("Written %d bytes to %s", len(content), path)}
}

func runShell(params map[string]string, timeoutSec int) Result {
	return runShellWithRTK(params["command"], timeoutSec)
}
func listFiles(params map[string]string, _ int) Result {
dir := params["directory"]
if dir == "" {
dir = params["path"]
}
if dir == "" {
dir = params["dir"]
}
if dir == "" {
dir = "."
}
ext := params["extension"]
var files []string
entries, err := os.ReadDir(dir)
if err != nil {
return Result{Success: false, Output: err.Error(), Error: "list_error"}
}
for _, d := range entries {
name := d.Name()
if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
continue
}
if ext != "" && filepath.Ext(name) != ext {
continue
}
if d.IsDir() {
files = append(files, name+"/")
} else {
files = append(files, name)
}
if len(files) > 200 {
break
}
}
if len(files) == 0 {
return Result{Success: true, Output: "(empty directory)"}
}
return Result{Success: true, Output: strings.Join(files, "\n")}
}

func searchFiles(params map[string]string, _ int) Result {
pattern := strings.ReplaceAll(params["pattern"], `"`, `\"`)
dir := params["directory"]
if dir == "" {
dir = params["path"]
}
if dir == "" {
dir = params["dir"]
}
if dir == "" {
dir = "."
}
cmd := exec.Command("grep", "-rn", "--include=*.ts", "--include=*.js", "--include=*.py", "--include=*.go", "--include=*.md", pattern, dir)
out, _ := cmd.Output()
output := strings.TrimSpace(string(out))
if output == "" {
return Result{Success: true, Output: "No matches found"}
}
lines := strings.Split(output, "\n")
if len(lines) > 30 {
output = strings.Join(lines[:30], "\n") + "\n... (truncated)"
}
if len(output) > MaxOutput {
output = output[:MaxOutput]
}
return Result{Success: true, Output: output}
}

func editFile(params map[string]string, _ int) Result {
	path := params["path"]
	oldText := params["old_text"]
	newText := params["new_text"]

	if path == "" {
		return Result{Success: false, Output: "path is required", Error: "missing_param"}
	}
	if oldText == "" {
		return Result{Success: false, Output: "old_text is required", Error: "missing_param"}
	}

	info, err := os.Stat(path)
	if err != nil {
		return Result{Success: false, Output: "File not found: " + path, Error: "not_found"}
	}
	mode := info.Mode()

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{Success: false, Output: err.Error(), Error: "read_error"}
	}
	content := string(data)

	count := strings.Count(content, oldText)
	if count == 0 {
		return Result{Success: false, Output: "old_text not found in file", Error: "no_match"}
	}
	if count > 1 {
		return Result{Success: false, Output: fmt.Sprintf("old_text found %d times (must be unique)", count), Error: "multiple_matches"}
	}

	newContent := strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(path, []byte(newContent), mode); err != nil {
		return Result{Success: false, Output: err.Error(), Error: "write_error"}
	}
	return Result{Success: true, Output: fmt.Sprintf("Edited %s: replaced %d bytes with %d bytes", path, len(oldText), len(newText))}
}

func globFiles(params map[string]string, _ int) Result {
	pattern := params["pattern"]
	dir := params["directory"]
	if dir == "" {
		dir = "."
	}

	var matches []string
	if strings.Contains(pattern, "**") {
		suffix := strings.TrimPrefix(pattern, "**/")
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			name := info.Name()
			if info.IsDir() && (name == ".git" || name == "node_modules") {
				return filepath.SkipDir
			}
			if !info.IsDir() {
				matched, _ := filepath.Match(suffix, name)
				if matched {
					matches = append(matches, path)
				}
			}
			if len(matches) > 200 {
				return fmt.Errorf("limit reached")
			}
			return nil
		})
		if err != nil && err.Error() != "limit reached" {
			return Result{Success: false, Output: err.Error(), Error: "walk_error"}
		}
	} else {
		fullPattern := filepath.Join(dir, pattern)
		var err error
		matches, err = filepath.Glob(fullPattern)
		if err != nil {
			return Result{Success: false, Output: err.Error(), Error: "glob_error"}
		}
	}

	if len(matches) == 0 {
		return Result{Success: true, Output: "No files matched"}
	}
	output := strings.Join(matches, "\n")
	if len(output) > MaxOutput {
		output = output[:MaxOutput] + "\n... (truncated)"
	}
	return Result{Success: true, Output: output}
}

func grepFiles(params map[string]string, _ int) Result {
	pattern := params["pattern"]
	dir := params["directory"]
	if dir == "" {
		dir = "."
	}
	fileType := params["file_type"]

	re, err := regexp.Compile(pattern)
	if err != nil {
		return Result{Success: false, Output: "Invalid regex: " + err.Error(), Error: "bad_pattern"}
	}

	var results []string
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := info.Name()
		if info.IsDir() && (name == ".git" || name == "node_modules" || name == "vendor") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		if fileType != "" {
			ext := strings.TrimPrefix(filepath.Ext(name), ".")
			if ext != fileType {
				return nil
			}
		}
		if info.Size() > 500_000 {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				results = append(results, fmt.Sprintf("%s:%d:%s", path, i+1, line))
				if len(results) > 50 {
					return fmt.Errorf("limit")
				}
			}
		}
		return nil
	})

	if len(results) == 0 {
		return Result{Success: true, Output: "No matches found"}
	}
	output := strings.Join(results, "\n")
	if len(output) > MaxOutput {
		output = output[:MaxOutput] + "\n... (truncated)"
	}
	return Result{Success: true, Output: output}
}

// FormatForPrompt returns tool descriptions for the system prompt.
// FormatForPrompt returns Markdown-formatted tool definitions for inclusion in a system prompt.
func FormatForPrompt() string {
var sb strings.Builder
for _, t := range Definitions {
sb.WriteString("### " + t.Name + "\n")
sb.WriteString(t.Description + "\nParameters:\n")
for _, p := range t.Params {
req := "required"
if !p.Required {
req = "optional"
}
sb.WriteString(fmt.Sprintf("  - %s (%s, %s): %s\n", p.Name, p.Type, req, p.Desc))
}
sb.WriteString("\n")
}
return sb.String()
}
