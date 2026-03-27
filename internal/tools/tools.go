package tools

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"

"github.com/AgentGuardHQ/shellforge/internal/governance"
"github.com/AgentGuardHQ/shellforge/internal/logger"
)

const MaxOutput = 8000

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
}

// Execute runs a tool call through governance, then executes if allowed.
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
ext := params["extension"]
var files []string
err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
if err != nil {
return nil
}
name := d.Name()
if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
if d.IsDir() {
return filepath.SkipDir
}
return nil
}
if len(files) > 200 {
return fmt.Errorf("limit reached")
}
if ext != "" && filepath.Ext(name) != ext {
return nil
}
rel, _ := filepath.Rel(".", path)
if d.IsDir() {
files = append(files, rel+"/")
} else {
files = append(files, rel)
}
return nil
})
if err != nil {
return Result{Success: false, Output: err.Error(), Error: "list_error"}
}
if len(files) == 0 {
return Result{Success: true, Output: "(empty directory)"}
}
return Result{Success: true, Output: strings.Join(files, "\n")}
}

func searchFiles(params map[string]string, _ int) Result {
pattern := strings.ReplaceAll(params["pattern"], `"`, `\"`)
dir := params["directory"]
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

// FormatForPrompt returns tool descriptions for the system prompt.
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
