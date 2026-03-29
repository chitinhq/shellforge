package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/AgentGuardHQ/shellforge/internal/governance"
	"github.com/AgentGuardHQ/shellforge/internal/ollama"
)

func cmdStatusFull() {
	fmt.Printf("ShellForge %s — Ecosystem Status\n", version)
	fmt.Println(strings.Repeat("─", 50))

	healthy := 0
	total := 0

	// ── Ollama ──
	total++
	fmt.Println("\n🦙 Ollama (local inference)")
	if ollama.IsRunning() {
		healthy++
		models, _ := ollama.ListModels()
		fmt.Printf("  ✓ running (%d models loaded)\n", len(models))
		for _, m := range models {
			tag := ""
			if m == ollama.Model {
				tag = " ← active"
			}
			fmt.Printf("    • %s%s\n", m, tag)
		}
	} else {
		fmt.Println("  ✗ not running")
		fmt.Println("    → ollama serve")
	}

	// ── Governance ──
	total++
	fmt.Println("\n🛡️  AgentGuard (governance)")
	configPath := findGovernanceConfig()
	if configPath != "" {
		eng, err := governance.NewEngine(configPath)
		if err != nil {
			fmt.Printf("  ✗ config error: %s\n", err)
		} else {
			healthy++
			fmt.Printf("  ✓ mode=%s, %d policies (%s)\n", eng.Mode, len(eng.Policies), configPath)
			for _, p := range eng.Policies {
				fmt.Printf("    • %s [%s]\n", p.Name, p.Action)
			}
		}
	} else {
		fmt.Println("  ✗ no agentguard.yaml found")
		fmt.Println("    → shellforge setup")
	}

	// ── Drivers ──
	fmt.Println("\n💻 Agent Drivers")
	drivers := []struct {
		name    string
		bin     string
		desc    string
		install string
	}{
		{"goose", "goose", "AI agent with native Ollama support (Block)", "brew install --cask block-goose"},
		{"claude", "claude", "Claude Code CLI", "npm i -g @anthropic-ai/claude-code"},
		{"copilot", "github-copilot-cli", "GitHub Copilot CLI", "gh extension install github/gh-copilot"},
		{"codex", "codex", "OpenAI Codex CLI", "npm i -g @openai/codex"},
		{"gemini", "gemini", "Google Gemini CLI", "npm i -g @anthropic-ai/gemini-cli"},
		{"openclaw", "openclaw", "OpenClaw browser automation (Anthropic)", "npm i -g @anthropic-ai/openclaw"},
		{"nemoclaw", "openclaw", "NemoClaw (OpenClaw + Nemotron sandbox)", "npm i -g @anthropic-ai/openclaw"},
	}
	driverCount := 0
	for _, d := range drivers {
		total++
		if _, err := exec.LookPath(d.bin); err == nil {
			healthy++
			driverCount++
			fmt.Printf("  ✓ %s: installed (%s)\n", d.name, d.desc)
		} else {
			fmt.Printf("  ○ %s: not found (%s)\n", d.name, d.install)
		}
	}
	if driverCount == 0 {
		fmt.Println("  → Install at least one driver, or use: shellforge agent \"prompt\"")
	}

	// ── Orchestration ──
	total++
	fmt.Println("\n📋 Dagu (orchestration)")
	if _, err := exec.LookPath("dagu"); err == nil {
		healthy++
		fmt.Println("  ✓ installed")
		fmt.Println("    → dagu server --dags=./dags   (web UI at :8080)")
	} else {
		fmt.Println("  ○ not installed")
		if isdarwin() {
			fmt.Println("    → brew install dagu")
		} else {
			fmt.Println("    → curl -sL https://raw.githubusercontent.com/dagu-org/dagu/main/scripts/installer.sh | bash")
		}
	}

	// ── RTK ──
	total++
	fmt.Println("\n⚡ RTK (token compression)")
	if _, err := exec.LookPath("rtk"); err == nil {
		healthy++
		fmt.Println("  ✓ installed (70-90% savings on shell output)")
	} else {
		fmt.Println("  ○ not installed (optional)")
		fmt.Println("    → npm i -g @anthropic/rtk")
	}

	// ── Docker / Sandbox ──
	total++
	fmt.Println("\n🔒 Docker (sandbox)")
	if _, err := exec.LookPath("docker"); err == nil {
		healthy++
		fmt.Println("  ✓ Docker available (sandbox-ready)")
	} else {
		fmt.Println("  ○ not installed (optional)")
		if isdarwin() {
			fmt.Println("    → brew install colima docker")
		} else {
			fmt.Println("    → curl -fsSL https://get.docker.com | sh")
		}
	}

	// ── DefenseClaw ──
	fmt.Println("\n🐾 DefenseClaw (supply chain scanner)")
	if _, err := exec.LookPath("defenseclaw"); err == nil {
		total++
		healthy++
		fmt.Println("  ✓ installed")
	} else {
		fmt.Println("  ○ not yet publicly available (Cisco AI Defense)")
		fmt.Println("    → https://github.com/cisco-ai-defense/defenseclaw")
	}

	// ── Summary ──
	fmt.Println("\n" + strings.Repeat("─", 50))
	fmt.Printf("Status: %d/%d components active\n", healthy, total)
	if healthy < 3 {
		fmt.Println("Run: shellforge setup")
	}
}

func isdarwin() bool {
	out, err := exec.Command("uname").Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "Darwin"
}
