<div align="center">

# ShellForge

**Governed local AI agents — one Go binary, zero cloud.**

[![Go](https://img.shields.io/badge/Go-1.18+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![GitHub Pages](https://img.shields.io/badge/Live_Site-agentguardhq.github.io/shellforge-ff6b2b?style=for-the-badge)](https://agentguardhq.github.io/shellforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![AgentGuard](https://img.shields.io/badge/Governed_by-AgentGuard-green?style=for-the-badge)](https://github.com/AgentGuardHQ/agentguard)

*Run autonomous AI agents on your machine with policy enforcement on every tool call. No cloud. No API keys. No data leaves your laptop.*

[Website](https://agentguardhq.github.io/shellforge) · [Docs](docs/architecture.md) · [Roadmap](docs/roadmap.md) · [AgentGuard](https://github.com/AgentGuardHQ/agentguard)

<img src="https://github.com/user-attachments/assets/a94a8a5e-dfeb-4771-a6ab-465d3c2f01f0" alt="ShellForge — Local Governed Agent Runtime" width="700">

</div>

---

## Quick Start

### Install via Homebrew (macOS / Linux)

```bash
brew tap AgentGuardHQ/tap
brew install shellforge

shellforge setup                 # pulls Ollama + model + verifies stack
shellforge run crush "analyze this repo for test gaps"
```

### Install from source

```bash
git clone https://github.com/AgentGuardHQ/shellforge.git
cd shellforge
go build -o shellforge ./cmd/shellforge/
./shellforge setup
```

**Requirements:** macOS (Apple Silicon/Intel) or Linux · ~1.3 GB RAM (1.7B model)

---

## What Is ShellForge?

ShellForge is a **governed agent runtime** — not an agent framework, not an orchestration layer, not a prompt wrapper.

It sits between any agent driver and the real world. The agent decides what it wants to do. ShellForge decides whether it's allowed.

```
Agent Driver (Crush, Claude Code, Copilot CLI)
  → ShellForge Governance (allow / deny / correct)
    → Your Environment (files, shell, git)
```

**The core insight:** ShellForge's value is governance, not the agent loop. [Crush](https://github.com/charmbracelet/crush) handles agent execution. [Dagu](https://github.com/dagu-org/dagu) handles workflow orchestration. ShellForge wraps them all with [AgentGuard](https://github.com/AgentGuardHQ/agentguard) policy enforcement on every tool call.

---

## The Stack

| Layer | Project | What It Does |
|-------|---------|--------------|
| **Infer** | [Ollama](https://ollama.com) | Local LLM inference (Metal GPU on Mac) |
| **Optimize** | [RTK](https://github.com/rtk-ai/rtk) | Token compression — 70-90% reduction on shell output |
| **Execute** | [Crush](https://github.com/charmbracelet/crush) | Go-native AI coding agent (TUI + headless) |
| **Orchestrate** | [Dagu](https://github.com/dagu-org/dagu) | YAML DAG workflows with scheduling and web UI |
| **Govern** | [AgentGuard](https://github.com/AgentGuardHQ/agentguard) | Policy enforcement on every action — allow/deny/correct |
| **Sandbox** | [OpenShell](https://github.com/NVIDIA/OpenShell) | Kernel-level isolation (Docker on macOS) |
| **Scan** | [DefenseClaw](https://github.com/cisco-ai-defense/defenseclaw) | Supply chain scanner — AI Bill of Materials |

```bash
shellforge status
# Ollama        running (qwen3:30b loaded)
# RTK           v0.4.2
# Crush         v1.0.0
# AgentGuard    enforce mode (5 rules)
# Dagu          connected (web UI at :8080)
# OpenShell     Docker sandbox active
# DefenseClaw   scanner ready
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `shellforge run crush "prompt"` | Run Crush with full governance |
| `shellforge serve agents.yaml` | Daemon mode — run scheduled agent swarm |
| `shellforge agent "prompt"` | Run built-in agent with governance |
| `shellforge status` | Show ecosystem health |
| `shellforge setup` | Install Ollama, pull model, verify stack |
| `shellforge scan` | Run security scan via DefenseClaw |
| `shellforge version` | Print version |

---

## Architecture

```
┌───────────────────────────────────────────────────┐
│  Dagu (Orchestration)                              │
│  YAML DAGs · Cron scheduling · Web UI · Retries    │
└────────────────────┬──────────────────────────────┘
                     │ task
┌────────────────────▼──────────────────────────────┐
│  Crush (Execution Engine)                          │
│  Agent loop · Tool calling · TUI + headless        │
│  Uses Ollama for inference                         │
└────────────────────┬──────────────────────────────┘
                     │ tool call
          ═══════════╪═══════════
          ║  AgentGuard          ║
          ║  Governance Kernel   ║
          ║  allow · deny · audit║
          ║  every. single. call.║
          ═══════════╪═══════════
                     │ approved
┌────────────────────▼──────────────────────────────┐
│  Your Environment                                  │
│  Files · Shell (RTK) · Git · Network               │
│  Sandboxed by OpenShell                            │
└───────────────────────────────────────────────────┘
```

---

## Governance

ShellForge's core value. Every tool call passes through `agentguard.yaml` before execution.

```yaml
# agentguard.yaml — policy-as-code for every agent action
mode: enforce  # enforce | monitor

policies:
  - name: no-force-push
    action: deny
    pattern: "git push --force"

  - name: no-destructive-rm
    action: deny
    pattern: "rm -rf"

  - name: no-secret-access
    action: deny
    pattern: "*.env|*id_rsa|*id_ed25519"
```

When an action is denied, ShellForge's **correction engine** feeds structured feedback back to the model so it can self-correct — not just fail.

---

## Swarm Mode

Run a 24/7 agent swarm on your Mac with memory-aware scheduling:

```bash
shellforge serve agents.yaml
```

Auto-detects RAM, calculates max parallel Ollama slots, queues the rest.

```yaml
# agents.yaml
max_parallel: 0     # 0 = auto-detect from RAM
model_ram_gb: 19    # qwen3:30b Q4

agents:
  - name: qa-agent
    system: "You are a QA engineer."
    prompt: "Analyze the repo for test gaps."
    schedule: "4h"
    priority: 2
    timeout: 300
    enabled: true
```

**Memory budget (qwen3:30b Q4):**

| Mac | RAM | Free for KV | Max Parallel |
|-----|-----|-------------|--------------|
| M4 Pro 48GB | 48 GB | ~25 GB | 3-4 agents |
| M4 32GB | 32 GB | ~9 GB | 1-2 agents |

**Tip:** `OLLAMA_KV_CACHE_TYPE=q8_0` halves KV cache memory — doubles agent capacity.

---

## Model Options

| Model | Params | RAM | Best For |
|-------|--------|-----|----------|
| `qwen3:1.7b` | 1.7B | ~1.2 GB | Fast tasks, prototyping |
| `qwen3:4b` | 4B | ~3 GB | Balanced reasoning |
| `qwen3:30b` | 30B | ~19 GB | Production quality (M4 Pro 48GB) |
| `mistral:7b` | 7B | ~5 GB | Complex analysis |

---

## macOS (Apple Silicon / M4)

- **Ollama** uses Metal GPU acceleration — no CUDA needed
- **KV cache quantization** (`OLLAMA_KV_CACHE_TYPE=q8_0`) halves memory per agent slot
- **OpenShell** requires Docker via [Colima](https://github.com/abiosoft/colima)

---

## The AgentGuard Platform

| Project | What It Does |
|---------|--------------|
| [**AgentGuard**](https://github.com/AgentGuardHQ/agentguard) | Governance kernel — policy enforcement for any agent driver |
| [**AgentGuard Cloud**](https://github.com/AgentGuardHQ/agentguard-cloud) | SaaS dashboard — observability, session replay, compliance |
| **ShellForge** | Governed local agent runtime — the onramp to AgentGuard |

---

## Contributing

```bash
git checkout -b feat/my-feature
go build ./cmd/shellforge/
go test ./...
```

See [docs/roadmap.md](docs/roadmap.md) for what's planned.

---

<div align="center">

**[Website](https://agentguardhq.github.io/shellforge)** · **[Star on GitHub](https://github.com/AgentGuardHQ/shellforge)** · **[AgentGuard](https://github.com/AgentGuardHQ/agentguard)**

*Built by humans and agents*

MIT License

</div>
