// ShellForge — local governed agent runtime.
// Single Go binary. Wraps Ollama + governance. Full ecosystem integration.
package main

import (
"bufio"
"encoding/json"
"fmt"
"io"
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

var version = "0.4.8"

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
case "run":
if len(os.Args) < 3 {
fmt.Fprintln(os.Stderr, "Usage: shellforge run <driver> \"prompt\"")
fmt.Fprintln(os.Stderr, "Drivers: goose, claude, copilot, codex, gemini, openclaw, nemoclaw")
os.Exit(1)
}
driver := os.Args[2]
prompt := ""
if len(os.Args) > 3 {
prompt = strings.Join(os.Args[3:], " ")
}
cmdRun(driver, prompt)
case "evaluate":
cmdEvaluate()
case "agent":
if len(os.Args) < 3 {
fmt.Fprintln(os.Stderr, "Usage: shellforge agent \"your prompt\"")
os.Exit(1)
}
cmdAgent(strings.Join(os.Args[2:], " "))
case "swarm":
cmdSwarm()
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
  shellforge run <driver> "prompt"  Run a governed agent (claude, copilot, codex, gemini, openclaw, nemoclaw)
  shellforge setup                  Install Ollama, pull model, verify stack
  shellforge qa [target]            QA analysis with tool use + governance
  shellforge report [repo]          Weekly status report from git + logs
  shellforge agent "prompt"         Run any task with agentic tool use
  shellforge status                 Full ecosystem health check
  shellforge scan [dir]             DefenseClaw supply chain scan
  shellforge version                Print version

  shellforge serve [config]        Simple daemon mode (built-in scheduler)
  shellforge swarm                 Setup Dagu orchestration (DAG workflows + web UI)

Governance:  agentguard.yaml — every tool call evaluated before execution.
Stack:       Ollama · AgentGuard · Dagu · RTK

`, version)
}

func cmdSetup() {
fmt.Println()
fmt.Println("╔══════════════════════════════════════╗")
fmt.Println("║     ShellForge Setup Wizard          ║")
fmt.Println("╚══════════════════════════════════════╝")
fmt.Println()

reader := bufio.NewReader(os.Stdin)
steps := 0
total := 6

// ── Detect environment ──
isServer := !hasGPU() && runtime.GOOS == "linux"
model := ""

// ── Step 1: Ollama (skip on headless server) ──
steps++
if isServer {
fmt.Printf("── Step %d/%d: Ollama (skipped — server mode) ──\n", steps, total)
fmt.Println("  Detected: Linux, no GPU — skipping local model setup")
fmt.Println("  Use CLI drivers instead: shellforge run claude, copilot, codex, gemini")
fmt.Println()
} else {
fmt.Printf("── Step %d/%d: Ollama (local LLM inference) ──\n", steps, total)
if _, err := exec.LookPath("ollama"); err != nil {
fmt.Print("  Ollama not found. Install? [Y/n] ")
if confirm(reader) {
fmt.Println("  → Installing Ollama...")
if runtime.GOOS == "darwin" {
run("brew", "install", "ollama")
} else {
run("sh", "-c", "curl -fsSL https://ollama.ai/install.sh | sh")
}
}
} else {
fmt.Println("  ✓ Ollama installed")
}

if !ollama.IsRunning() {
fmt.Println("  → Starting Ollama...")
cmd := exec.Command("ollama", "serve")
cmd.Start()
time.Sleep(3 * time.Second)
}
if ollama.IsRunning() {
fmt.Println("  ✓ Ollama running")
} else {
fmt.Println("  ⚠ Ollama not responding")
fmt.Println("    Start it manually: ollama serve")
fmt.Println("    Then re-run: shellforge setup")
}

// Pick model
fmt.Println()
fmt.Println("  Available models:")
fmt.Println("    1) qwen3:1.7b  — 1.2 GB RAM, fastest")
fmt.Println("    2) qwen3:8b    — 6 GB RAM, balanced (recommended)")
fmt.Println("    3) qwen3:30b   — 19 GB RAM, best quality (M4 Pro 48GB+)")
fmt.Println("    4) phi4         — 9 GB RAM, Microsoft")
fmt.Println("    5) other       — enter a custom model name")
fmt.Print("  Pick model [2]: ")
choice := readLine(reader)
model = "qwen3:8b"
switch strings.TrimSpace(choice) {
case "1":
model = "qwen3:1.7b"
case "2", "":
model = "qwen3:8b"
case "3":
model = "qwen3:30b"
case "4":
model = "phi4"
case "5":
fmt.Print("  Model name: ")
model = readLine(reader)
default:
model = strings.TrimSpace(choice)
}
fmt.Printf("  → Pulling %s (this may take a few minutes)...\n", model)
run("ollama", "pull", model)
fmt.Printf("  ✓ Model ready: %s\n", model)

if model != ollama.Model {
fmt.Printf("  Note: set OLLAMA_MODEL=%s before running shellforge\n", model)
}
fmt.Println()
}

// ── Step 2: Governance ──
steps++
fmt.Printf("── Step %d/%d: Governance (agentguard.yaml) ──\n", steps, total)
configPath := findGovernanceConfig()
if configPath == "" {
fmt.Print("  No agentguard.yaml found. Create default? [Y/n] ")
if confirm(reader) {
writeDefaultGovernanceConfig()
configPath = "agentguard.yaml"
}
}
if configPath != "" {
eng, err := governance.NewEngine(configPath)
if err != nil {
fmt.Printf("  ⚠ Config error: %s\n", err)
} else {
fmt.Printf("  ✓ Governance: mode=%s, %d policies\n", eng.Mode, len(eng.Policies))
}
}
fmt.Println()

// ── Step 3: Dagu (orchestration) ──
steps++
fmt.Printf("── Step %d/%d: Dagu (orchestration + web UI) ──\n", steps, total)
if _, err := exec.LookPath("dagu"); err != nil {
fmt.Print("  Dagu not found. Install? [Y/n] ")
if confirm(reader) {
fmt.Println("  → Installing Dagu...")
if runtime.GOOS == "darwin" {
run("brew", "install", "dagu")
} else {
run("sh", "-c", "curl -sL https://raw.githubusercontent.com/dagu-org/dagu/main/scripts/installer.sh | bash")
}
} else {
fmt.Println("  Skipped (install later: brew install dagu)")
}
} else {
fmt.Println("  ✓ Dagu installed")
}

// Create example DAGs
if _, err := os.Stat("dags"); os.IsNotExist(err) {
fmt.Print("  Create example swarm workflows? [Y/n] ")
if confirm(reader) {
os.MkdirAll("dags", 0o755)
writeExampleDAGs()
fmt.Println("  ✓ dags/sdlc-swarm.yaml created")
}
} else {
entries, _ := filepath.Glob("dags/*.yaml")
fmt.Printf("  ✓ dags/ exists (%d workflows)\n", len(entries))
}
fmt.Println()

// ── Step 4: RTK (optional) ──
steps++
fmt.Printf("── Step %d/%d: RTK (token compression, optional) ──\n", steps, total)
if _, err := exec.LookPath("rtk"); err != nil {
fmt.Print("  RTK not found. Install? [y/N] ")
input := readLine(reader)
if strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
fmt.Println("  → Installing RTK...")
run("npm", "i", "-g", "@anthropic/rtk")
} else {
fmt.Println("  Skipped (saves 70-90% tokens on shell output)")
}
} else {
fmt.Println("  ✓ RTK installed")
}
fmt.Println()

// ── Step 5: Agent drivers ──
steps++
fmt.Printf("── Step %d/%d: Agent drivers ──\n", steps, total)

// On Mac/GPU: offer Goose (local models via Ollama). On server: skip, show API drivers.
if !isServer {
if _, err := exec.LookPath("goose"); err != nil {
fmt.Println("  Goose — AI agent with native Ollama support (actually executes tools)")
fmt.Print("  Install Goose? [Y/n] ")
if confirm(reader) {
fmt.Println("  → Installing Goose...")
if runtime.GOOS == "darwin" {
run("brew", "install", "--cask", "block-goose")
} else {
run("sh", "-c", "curl -fsSL https://github.com/block/goose/releases/download/stable/download_cli.sh | bash")
}
if _, err := exec.LookPath("goose"); err == nil {
fmt.Println("  ✓ Goose installed")
fmt.Println("  → Run 'goose configure' to set up Ollama provider")
} else {
fmt.Println("  ⚠ Install failed — try: brew install --cask block-goose")
}
}
} else {
fmt.Println("  ✓ Goose installed (local model driver)")
}
}

// Show API-based drivers
apiDrivers := []struct {
name    string
bin     string
install string
}{
{"Claude Code", "claude", "npm i -g @anthropic-ai/claude-code"},
{"Copilot CLI", "github-copilot-cli", "gh extension install github/gh-copilot"},
{"Codex CLI", "codex", "npm i -g @openai/codex"},
{"Gemini CLI", "gemini", "npm i -g @google/gemini-cli"},
}
fmt.Println()
fmt.Println("  API-based drivers (use their own model APIs):")
installedDrivers := 0
for _, d := range apiDrivers {
if _, err := exec.LookPath(d.bin); err == nil {
fmt.Printf("  ✓ %s installed\n", d.name)
installedDrivers++
} else {
fmt.Printf("  ○ %s → %s\n", d.name, d.install)
}
}
if isServer && installedDrivers == 0 {
fmt.Println()
fmt.Println("  ⚠ No drivers installed. Install at least one:")
fmt.Println("    npm i -g @anthropic-ai/claude-code")
}
fmt.Println()

// ── Step 6: Docker sandbox (optional) ──
steps++
fmt.Printf("── Step %d/%d: Docker sandbox (optional) ──\n", steps, total)
if _, err := exec.LookPath("docker"); err == nil {
fmt.Println("  ✓ Docker available (sandbox-ready)")
} else {
fmt.Print("  Docker (for sandboxed agent execution)? [y/N] ")
input := readLine(reader)
if strings.HasPrefix(strings.ToLower(strings.TrimSpace(input)), "y") {
if runtime.GOOS == "darwin" {
fmt.Println("  → Installing Colima + Docker...")
run("brew", "install", "colima", "docker")
fmt.Println("  → Starting Colima...")
run("colima", "start")
if _, err := exec.LookPath("docker"); err == nil {
fmt.Println("  ✓ Docker ready")
} else {
fmt.Println("  ⚠ Docker not available after install — check: colima status")
}
} else {
fmt.Println("  → Installing Docker...")
run("sh", "-c", "curl -fsSL https://get.docker.com | sh")
if _, err := exec.LookPath("docker"); err == nil {
fmt.Println("  ✓ Docker installed")
} else {
fmt.Println("  ⚠ Install failed — try: https://docs.docker.com/engine/install/")
}
}
} else {
fmt.Println("  Skipped")
}
}
fmt.Println()

// ── Output dirs ──
os.MkdirAll("outputs/logs", 0o755)
os.MkdirAll("outputs/reports", 0o755)

// ── Summary ──
fmt.Println("╔══════════════════════════════════════╗")
fmt.Println("║     Setup Complete                   ║")
fmt.Println("╚══════════════════════════════════════╝")
fmt.Println()
if isServer {
fmt.Println("  Server mode — use CLI drivers:")
fmt.Println("    shellforge run claude \"review open PRs\"")
fmt.Println("    shellforge run copilot \"update docs\"")
fmt.Println("    shellforge run codex \"generate tests\"")
fmt.Println()
fmt.Println("  Run a swarm:")
fmt.Println("    shellforge swarm                      # start Dagu dashboard")
fmt.Println("    dagu start dags/multi-driver-swarm.yaml")
} else {
fmt.Println("  Quick start:")
fmt.Println("    shellforge run goose \"describe this project\"")
fmt.Println("    goose configure                       # set up Ollama if not done")
fmt.Println()
fmt.Println("  Run a swarm:")
fmt.Println("    shellforge swarm                      # start Dagu dashboard")
fmt.Println("    dagu start dags/sdlc-swarm.yaml")
if model != "" {
fmt.Printf("\n  Tip: export OLLAMA_MODEL=%s\n", model)
}
fmt.Println("  Tip: export OLLAMA_KV_CACHE_TYPE=q8_0   # halves memory per agent")
}
fmt.Println()
}

func confirm(r *bufio.Reader) bool {
input := readLine(r)
trimmed := strings.TrimSpace(strings.ToLower(input))
return trimmed == "" || trimmed == "y" || trimmed == "yes"
}

func readLine(r *bufio.Reader) string {
line, _ := r.ReadString('\n')
return strings.TrimRight(line, "\r\n")
}

func writeDefaultGovernanceConfig() {
policy := `# agentguard.yaml — ShellForge governance policy
# Mode: enforce (block violations) or monitor (log only)
mode: enforce

policies:
  - name: no-force-push
    description: Prevent force pushes to any remote
    match:
      command: "git push"
      args_contain: ["--force", "-f", "--force-with-lease"]
    action: deny
    message: "Force push is not allowed by governance policy"

  - name: no-destructive-rm
    description: Block recursive force deletion
    match:
      command: "rm"
      args_contain: ["-rf", "-fr"]
    action: deny
    message: "Recursive force deletion is not allowed"

  - name: no-secret-access
    description: Prevent reading sensitive files
    match:
      path_pattern: "*.env|*id_rsa|*id_ed25519|*.pem"
    action: deny
    message: "Access to secrets/keys is not allowed"
`
if err := os.WriteFile("agentguard.yaml", []byte(policy), 0o644); err != nil {
fmt.Printf("  ⚠ Could not create agentguard.yaml: %s\n", err)
} else {
fmt.Println("  ✓ agentguard.yaml created (enforce mode, 3 policies)")
}
}

func writeExampleDAGs() {
dag := `# sdlc-swarm.yaml — Daily SDLC agent swarm
schedule: "0 9 * * *"

steps:
  - name: qa-analysis
    command: shellforge agent "Analyze source code for test gaps and quality issues. Use read_file and list_files. Produce a structured report."

  - name: security-scan
    command: shellforge agent "Check for exposed secrets, insecure dependencies, and misconfigurations."
    depends:
      - qa-analysis

  - name: daily-report
    command: shellforge agent "Generate a daily status report from git log and previous agent findings."
    depends:
      - qa-analysis
      - security-scan
`
os.WriteFile("dags/sdlc-swarm.yaml", []byte(dag), 0o644)
}

// ── Driver configuration ──

type driverConfig struct {
	binary       string   // CLI binary name
	buildCmd     func(string) []string // build command args from prompt
	interactive  []string // args when no prompt given
	hasHooks     bool     // whether AgentGuard hooks are configured for this driver
	initHint     string   // command to set up hooks
}

var drivers = map[string]driverConfig{
	"claude": {
		binary:   "claude",
		buildCmd: func(p string) []string { return []string{"--dangerously-skip-permissions", "-p", p} },
		interactive: []string{},
		hasHooks:    true,
		initHint:    "agentguard claude-init",
	},
	"copilot": {
		binary:   "copilot-cli",
		buildCmd: func(p string) []string { return []string{"--prompt", p} },
		interactive: []string{},
		hasHooks:    true,
		initHint:    "agentguard copilot-init",
	},
	"codex": {
		binary:   "codex",
		buildCmd: func(p string) []string { return []string{"--quiet", "--prompt", p} },
		interactive: []string{},
		hasHooks:    true,
		initHint:    "agentguard codex-init",
	},
	"gemini": {
		binary:   "gemini",
		buildCmd: func(p string) []string { return []string{"--prompt", p} },
		interactive: []string{},
		hasHooks:    false,
		initHint:    "agentguard gemini-init",
	},
	"goose": {
		binary: "goose",
		buildCmd: func(p string) []string {
			return []string{"run", "--no-session", "-t", p}
		},
		interactive: []string{},
		hasHooks:    false,
		initHint:    "",
	},
	"openclaw": {
		binary: "openclaw",
		buildCmd: func(p string) []string {
			return []string{"--non-interactive", "-p", p}
		},
		interactive: []string{},
		hasHooks:    false,
		initHint:    "agentguard openclaw-init",
	},
	"nemoclaw": {
		binary: "openclaw",
		buildCmd: func(p string) []string {
			return []string{"--non-interactive", "--model", "nemotron", "--sandbox", "openshell", "-p", p}
		},
		interactive: []string{},
		hasHooks:    false,
		initHint:    "agentguard nemoclaw-init",
	},
}

func cmdRun(driver, prompt string) {
	dc, ok := drivers[driver]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown driver: %s\n", driver)
		fmt.Fprintln(os.Stderr, "Available drivers: claude, copilot, codex, gemini, openclaw, nemoclaw")
		os.Exit(1)
	}

	// Check driver CLI is installed
	if _, err := exec.LookPath(dc.binary); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s not found in PATH\n", dc.binary)
		fmt.Fprintf(os.Stderr, "Install %s and try again.\n", dc.binary)
		os.Exit(1)
	}

	// Check governance config exists
	configPath := findGovernanceConfig()
	if configPath == "" {
		fmt.Fprintln(os.Stderr, "ERROR: agentguard.yaml not found")
		fmt.Fprintln(os.Stderr, "Run: shellforge setup")
		os.Exit(1)
	}

	// Warn if hooks not configured for this driver
	if !dc.hasHooks {
		fmt.Fprintf(os.Stderr, "WARNING: Governance hooks not configured for %s. Run: %s\n", driver, dc.initHint)
	}

	// Build command args
	var args []string
	if prompt != "" {
		args = dc.buildCmd(prompt)
	} else {
		args = dc.interactive
	}

	// Log the run
	ts := time.Now().Format(time.RFC3339)
	fmt.Printf("[shellforge] %s — driver=%s prompt=%q\n", ts, driver, prompt)

	// Spawn driver as subprocess with passthrough I/O
	cmd := exec.Command(dc.binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	// For Goose/OpenClaw/NemoClaw: set governed shell so ALL commands go through AgentGuard
	if driver == "goose" || driver == "openclaw" || driver == "nemoclaw" {
		sfBin, _ := exec.LookPath("shellforge")
		if sfBin != "" {
			// Find govern-shell.sh next to the shellforge binary or in known locations
			governShell := ""
			for _, path := range []string{
				filepath.Join(filepath.Dir(sfBin), "..", "share", "shellforge", "govern-shell.sh"),
				filepath.Join(filepath.Dir(sfBin), "govern-shell.sh"),
				"scripts/govern-shell.sh",
				"/usr/local/share/shellforge/govern-shell.sh",
			} {
				if _, err := os.Stat(path); err == nil {
					governShell = path
					break
				}
			}
			if governShell != "" {
				cmd.Env = append(cmd.Env, "SHELL="+governShell)
				cmd.Env = append(cmd.Env, "SHELLFORGE_REAL_SHELL=/bin/bash")
				fmt.Println("[shellforge] governance: shell wrapper active — every command evaluated")
			}
		}
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
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

func cmdSwarm() {
fmt.Println("=== ShellForge Swarm Setup (Dagu) ===")
fmt.Println()

// Check if Dagu is installed
if _, err := exec.LookPath("dagu"); err != nil {
fmt.Println("Dagu is not installed. Install it:")
fmt.Println()
if runtime.GOOS == "darwin" {
fmt.Println("  brew install dagu")
} else {
fmt.Println("  curl -sL https://raw.githubusercontent.com/dagu-org/dagu/main/scripts/installer.sh | bash")
}
fmt.Println()
fmt.Println("Then run 'shellforge swarm' again.")
return
}
fmt.Println("✓ Dagu installed")

// Check for dags directory
if _, err := os.Stat("dags"); os.IsNotExist(err) {
fmt.Println("→ Creating dags/ directory with example workflows...")
os.MkdirAll("dags", 0o755)

// Write SDLC swarm DAG
sdlcDAG := `# sdlc-swarm.yaml — Daily SDLC agent swarm
schedule: "0 9 * * *"

steps:
  - name: qa-analysis
    command: shellforge agent "Analyze source code for test gaps and quality issues. Use read_file and list_files. Produce a structured report."

  - name: security-scan
    command: shellforge agent "Check for exposed secrets, insecure dependencies, and misconfigurations."
    depends:
      - qa-analysis

  - name: daily-report
    command: shellforge agent "Generate a daily status report from git log and previous agent findings."
    depends:
      - qa-analysis
      - security-scan
`
os.WriteFile("dags/sdlc-swarm.yaml", []byte(sdlcDAG), 0o644)
fmt.Println("  ✓ dags/sdlc-swarm.yaml (QA → security → report)")
} else {
fmt.Println("✓ dags/ directory exists")
// Count DAGs
entries, _ := filepath.Glob("dags/*.yaml")
fmt.Printf("  %d DAG(s) found\n", len(entries))
}

// Check for governance
configPath := findGovernanceConfig()
if configPath == "" {
fmt.Println("⚠ No agentguard.yaml — run 'shellforge setup' first")
return
}
fmt.Println("✓ Governance config found")

// Get absolute path to dags directory
dagsDir, _ := filepath.Abs("dags")

// Start Dagu server
fmt.Println()
fmt.Printf("→ Starting Dagu server (dags: %s)\n", dagsDir)
fmt.Println("  Dashboard: http://localhost:8080")
fmt.Println("  Press Ctrl+C to stop")
fmt.Println()

cmd := exec.Command("dagu", "server", "--dags="+dagsDir)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
cmd.Stdin = os.Stdin
cmd.Run()
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

func cmdEvaluate() {
// Read JSON from stdin, evaluate against governance, output JSON result.
// Used by Crush fork to check actions before execution.
data, err := io.ReadAll(os.Stdin)
if err != nil {
json.NewEncoder(os.Stdout).Encode(map[string]any{"allowed": false, "reason": "stdin read error"})
return
}

var input struct {
Tool   string `json:"tool"`
Action string `json:"action"`
Path   string `json:"path"`
}
if err := json.Unmarshal(data, &input); err != nil {
json.NewEncoder(os.Stdout).Encode(map[string]any{
"allowed": false,
"reason":  "malformed governance request: " + err.Error(),
})
return
}

configPath := findGovernanceConfig()
if configPath == "" {
json.NewEncoder(os.Stdout).Encode(map[string]any{"allowed": true, "reason": "no policy"})
return
}

engine, err := governance.NewEngine(configPath)
if err != nil {
json.NewEncoder(os.Stdout).Encode(map[string]any{"allowed": true, "reason": "policy error"})
return
}

// Map Crush tool names to ShellForge tool names
tool := input.Tool
params := map[string]string{"command": input.Action, "path": input.Path}

decision := engine.Evaluate(tool, params)
json.NewEncoder(os.Stdout).Encode(map[string]any{
"allowed":    decision.Allowed,
"reason":     decision.Reason,
"policy":     decision.PolicyName,
"suggestion": "",
})
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

// hasGPU detects if the machine has a GPU (Metal on macOS, NVIDIA on Linux).
func hasGPU() bool {
if runtime.GOOS == "darwin" {
return true // All Macs have Metal GPU
}
// Linux: check for NVIDIA GPU
if _, err := exec.LookPath("nvidia-smi"); err == nil {
return true
}
// Check for render devices (AMD/Intel)
if _, err := os.Stat("/dev/dri/renderD128"); err == nil {
return true
}
return false
}
