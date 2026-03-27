// ShellForge — local governed agent runtime.
// Single Go binary. Wraps Ollama + governance. Full ecosystem integration.
package main

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"runtime"
"strings"
"time"

"github.com/AgentGuardHQ/shellforge/internal/agent"
"github.com/AgentGuardHQ/shellforge/internal/governance"
"github.com/AgentGuardHQ/shellforge/internal/logger"
"github.com/AgentGuardHQ/shellforge/internal/ollama"
"github.com/AgentGuardHQ/shellforge/internal/scheduler"
)

var version = "0.2.0"

func main() {
if len(os.Args) < 2 {
printUsage()
os.Exit(1)
}

switch os.Args[1] {
case "setup":
cmdSetup()
case "qa":
target := "."
if len(os.Args) > 2 {
target = os.Args[2]
}
cmdQA(target)
case "report":
repo := "."
if len(os.Args) > 2 {
repo = os.Args[2]
}
cmdReport(repo)
case "agent":
if len(os.Args) < 3 {
fmt.Fprintln(os.Stderr, "Usage: shellforge agent \"your prompt\"")
os.Exit(1)
}
cmdAgent(strings.Join(os.Args[2:], " "))
case "serve":
configPath := "agents.yaml"
if len(os.Args) > 2 {
configPath = os.Args[2]
}
cmdServe(configPath)
case "status":
cmdStatusFull()
case "scan":
cmdScan()
case "version", "--version", "-v":
fmt.Printf("shellforge %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
case "help", "--help", "-h":
printUsage()
default:
fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
printUsage()
os.Exit(1)
}
}

func printUsage() {
fmt.Printf(`ShellForge %s — local governed agent runtime

Usage:
  shellforge setup                 Install Ollama, pull model, verify stack
  shellforge qa [target]           QA analysis with tool use + governance
  shellforge report [repo]         Weekly status report from git + logs
  shellforge agent "prompt"        Run any task with agentic tool use
  shellforge status                Full ecosystem health check
  shellforge scan [dir]            DefenseClaw supply chain scan
  shellforge version               Print version

  shellforge serve [config]       Daemon mode — run scheduled agent swarm
Governance:  agentguard.yaml — every tool call evaluated before execution.
Engines:     OpenCode · DeepAgents · Paperclip
Stack:       RTK · TurboQuant · Ollama · AgentGuard · OpenShell · DefenseClaw

`, version)
}

func cmdSetup() {
fmt.Println("=== ShellForge Setup ===")
fmt.Printf("✓ Go %s\n", runtime.Version())

if _, err := exec.LookPath("ollama"); err != nil {
fmt.Println("→ Installing Ollama...")
if runtime.GOOS == "darwin" {
run("brew", "install", "ollama")
} else {
run("sh", "-c", "curl -fsSL https://ollama.ai/install.sh | sh")
}
} else {
fmt.Println("✓ Ollama installed")
}

if !ollama.IsRunning() {
fmt.Println("→ Starting Ollama...")
cmd := exec.Command("ollama", "serve")
cmd.Start()
time.Sleep(3 * time.Second)
}
if ollama.IsRunning() {
fmt.Println("✓ Ollama running")
} else {
fmt.Println("⚠ Ollama not responding — start manually: ollama serve")
}

model := ollama.Model
fmt.Printf("→ Pulling model %s...\n", model)
run("ollama", "pull", model)
fmt.Printf("✓ Model ready: %s\n", model)

configPath := findGovernanceConfig()
if configPath != "" {
eng, err := governance.NewEngine(configPath)
if err != nil {
fmt.Printf("⚠ Governance config error: %s\n", err)
} else {
fmt.Printf("✓ Governance: mode=%s, %d policies\n", eng.Mode, len(eng.Policies))
}
} else {
fmt.Println("⚠ No agentguard.yaml found")
}

os.MkdirAll("outputs/logs", 0o755)
os.MkdirAll("outputs/reports", 0o755)
fmt.Println("✓ Output directories ready")
fmt.Println("=== ShellForge Setup Complete ===")
}

func cmdQA(target string) {
engine := mustGovernance()
mustOllama()

result, err := agent.RunLoop(agent.LoopConfig{
Agent:       "qa-agent",
System:      "You are a QA engineer. Analyze source code and suggest specific, actionable test cases. Use tools to read files and run type checks. Be thorough but concise.",
UserPrompt:  fmt.Sprintf("Analyze the source code in %q for test gaps and quality issues. Use list_files to find source files, read_file to examine them, and run_shell to run any available linters. Produce a structured report.", target),
Model:       ollama.Model,
MaxTurns:    10,
TimeoutMs:   120_000,
OutputDir:   "outputs/logs",
TokenBudget: 3000,
}, engine)
if err != nil {
logger.Error("qa-agent", err.Error())
os.Exit(1)
}
printResult("qa-agent", result)
saveReport("outputs/logs", "qa", result)
}

func cmdReport(repo string) {
engine := mustGovernance()
mustOllama()

result, err := agent.RunLoop(agent.LoopConfig{
Agent:       "report-agent",
System:      "You are a technical writer. Generate concise markdown status reports. Use tools to gather data.",
UserPrompt:  fmt.Sprintf("Generate a weekly status report for %q. Use run_shell for git log, list_files + read_file for agent logs in outputs/. Output markdown with Summary, Changes, Recommendations.", repo),
Model:       ollama.Model,
MaxTurns:    8,
TimeoutMs:   90_000,
OutputDir:   "outputs/reports",
TokenBudget: 3000,
}, engine)
if err != nil {
logger.Error("report-agent", err.Error())
os.Exit(1)
}
printResult("report-agent", result)
saveReport("outputs/reports", "report", result)
}

func cmdAgent(prompt string) {
engine := mustGovernance()
mustOllama()

result, err := agent.RunLoop(agent.LoopConfig{
Agent:       "prototype-agent",
System:      "You are a senior engineer. Complete the requested task using available tools. Read files, write files, run commands, search code. Be precise.",
UserPrompt:  prompt,
Model:       ollama.Model,
MaxTurns:    15,
TimeoutMs:   180_000,
OutputDir:   "outputs/logs",
TokenBudget: 3000,
}, engine)
if err != nil {
logger.Error("prototype-agent", err.Error())
os.Exit(1)
}
printResult("prototype-agent", result)
saveReport("outputs/logs", "prototype", result)
}

func cmdServe(configPath string) {
engine := mustGovernance()
mustOllama()

cfg, err := scheduler.LoadConfig(configPath)
if err != nil {
fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
os.Exit(1)
}

fmt.Println("=== ShellForge Serve ===")
fmt.Printf("[serve] model: %s\n", ollama.Model)
fmt.Printf("[serve] governance: mode=%s, %d policies\n", engine.Mode, len(engine.Policies))
fmt.Printf("[serve] config: %s (%d agents)\n", configPath, len(cfg.Agents))

// Create runner that uses the existing agent.RunLoop
run := func(name, system, prompt string, timeoutSec int) error {
result, err := agent.RunLoop(agent.LoopConfig{
Agent:       name,
System:      system,
UserPrompt:  prompt,
Model:       ollama.Model,
MaxTurns:    12,
TimeoutMs:   timeoutSec * 1000,
OutputDir:   cfg.LogDir,
TokenBudget: 3000,
}, engine)
if err != nil {
logger.Error(name, err.Error())
return err
}
printResult(name, result)
saveReport(cfg.LogDir, name, result)
return nil
}

sched := scheduler.New(cfg, run)
sched.Start()
fmt.Println("[serve] running — press Ctrl+C to stop")
sched.Wait()
}

func cmdScan() {
fmt.Println("[🐾 DefenseClaw] Scanning agent skills and plugins...")
dir := "."
if len(os.Args) > 2 {
dir = os.Args[2]
}
// Use defenseclaw if available, otherwise do a basic scan
if _, err := exec.LookPath("defenseclaw"); err == nil {
cmd := exec.Command("defenseclaw", "scan", "--dir", dir)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Run()
} else {
fmt.Println("  DefenseClaw not installed — running basic integrity check")
fmt.Printf("  Scanning %s for agent configs...\n", dir)
// Basic: check for agentguard.yaml, list agent files
if _, err := os.Stat("agentguard.yaml"); err == nil {
fmt.Println("  ✓ agentguard.yaml found")
}
entries, _ := filepath.Glob(filepath.Join(dir, "agents", "*.ts"))
goEntries, _ := filepath.Glob(filepath.Join(dir, "internal", "**", "*.go"))
fmt.Printf("  Found %d TS agents, %d Go files\n", len(entries), len(goEntries))
fmt.Println("  Install defenseclaw for full supply chain scanning")
}
}

// ── Helpers ──

func mustGovernance() *governance.Engine {
configPath := findGovernanceConfig()
if configPath == "" {
fmt.Fprintln(os.Stderr, "ERROR: agentguard.yaml not found")
os.Exit(1)
}
eng, err := governance.NewEngine(configPath)
if err != nil {
fmt.Fprintf(os.Stderr, "ERROR: governance config: %s\n", err)
os.Exit(1)
}
fmt.Printf("[🛡️ AgentGuard] governance loaded — mode: %s, %d policies\n", eng.Mode, len(eng.Policies))
return eng
}

func mustOllama() {
if !ollama.IsRunning() {
fmt.Fprintln(os.Stderr, "ERROR: Ollama not running. Start: ollama serve")
os.Exit(1)
}
}

func findGovernanceConfig() string {
for _, c := range []string{"agentguard.yaml", filepath.Join("..", "agentguard.yaml")} {
if _, err := os.Stat(c); err == nil {
return c
}
}
return ""
}

func printResult(name string, r *agent.RunResult) {
fmt.Println()
status := "✓ success"
if !r.Success {
status = "✗ failed"
}
fmt.Printf("[%s] %s — %d turns, %d tool calls, %d denials\n", name, status, r.Turns, r.ToolCalls, r.Denials)
fmt.Printf("  tokens: %d prompt + %d response | %dms\n", r.PromptTok, r.ResponseTok, r.DurationMs)
if r.Output != "" {
fmt.Println()
fmt.Println(r.Output)
}
}

func saveReport(dir, prefix string, r *agent.RunResult) {
os.MkdirAll(dir, 0o755)
ts := time.Now().Format("2006-01-02T15-04-05")
path := filepath.Join(dir, fmt.Sprintf("%s-%s.md", prefix, ts))
content := fmt.Sprintf("# %s — %s\n\n**Turns:** %d | **Tool calls:** %d | **Denials:** %d\n**Tokens:** %d+%d | **Duration:** %dms\n\n%s\n",
prefix, time.Now().Format(time.RFC3339), r.Turns, r.ToolCalls, r.Denials, r.PromptTok, r.ResponseTok, r.DurationMs, r.Output)
os.WriteFile(path, []byte(content), 0o644)
fmt.Printf("\n→ Saved to %s\n", path)
}

func run(name string, args ...string) {
cmd := exec.Command(name, args...)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Run()
}
