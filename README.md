<div align="center">

# ShellForge

**Governed AI coding CLI and agent runtime — one Go binary, local or cloud.**

[![Go](https://img.shields.io/badge/Go-1.18+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev)
[![GitHub Pages](https://img.shields.io/badge/Live_Site-agentguardhq.github.io/shellforge-ff6b2b?style=for-the-badge)](https://agentguardhq.github.io/shellforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![AgentGuard](https://img.shields.io/badge/Governed_by-AgentGuard-green?style=for-the-badge)](https://github.com/chitinhq/agentguard)

*Interactive pair-programming with local models + autonomous multi-task execution — with governance on every tool call.*

[Website](https://agentguardhq.github.io/shellforge) · [Docs](docs/architecture.md) · [Roadmap](docs/roadmap.md) · [AgentGuard](https://github.com/chitinhq/agentguard)

<img src="https://github.com/user-attachments/assets/a94a8a5e-dfeb-4771-a6ab-465d3c2f01f0" alt="ShellForge — Local Governed Agent Runtime" width="700">

</div>

---

## Quick Start (Mac)

### 1. Install ShellForge

```bash
brew tap chitinhq/tap
brew install shellforge
```

Or from source: `git clone https://github.com/chitinhq/shellforge.git && cd shellforge && go build -o shellforge ./cmd/shellforge/`

### 2. Install Ollama (if you haven't already)

```bash
brew install ollama
ollama serve                     # start the model server (leave running)
```

### 3. Pull a model

```bash
ollama pull qwen3:8b             # 8B — good balance (needs ~6GB RAM)
# or: ollama pull qwen3:30b      # 30B — best quality (needs ~19GB, M4 Pro recommended)
# or: ollama pull qwen3:1.7b     # 1.7B — fastest, minimal RAM
```

### 4. Run setup inside any repo

```bash
cd ~/your-project                # navigate to any repo you want to work in
shellforge setup                 # creates agentguard.yaml + output dirs
```

This creates `agentguard.yaml` (governance policy) in your project root. Edit it to customize which actions are allowed/denied.

### 5. Start a chat session

```bash
shellforge chat                     # interactive REPL — pair-program with a local model
```

Or run a one-shot agent:

```bash
shellforge agent "describe what this project does"
shellforge agent "find test gaps and suggest improvements"
```

Every tool call (file reads, writes, shell commands) passes through governance before execution.

**Requirements:** macOS (Apple Silicon or Intel) or Linux

---

## What Is ShellForge?

ShellForge is a **governed AI coding CLI and agent runtime** — like Claude Code or Cursor, but with local models and policy enforcement built in.

Two modes:

1. **Interactive REPL** (`shellforge chat`) — pair-program with a local or cloud model. Persistent conversation history, shell escapes, color output.
2. **Autonomous agents** (`shellforge agent`, `shellforge ralph`) — one-shot tasks or multi-task loops with automatic validation and commit.

Both modes share the same governance layer. Every tool call passes through [AgentGuard](https://github.com/chitinhq/agentguard) policy enforcement before execution.

```
You (chat) or Octi Pulpo (dispatch)
  → ShellForge Agent Loop (tool calling, drift detection)
    → AgentGuard Governance (allow / deny / correct)
      → Your Environment (files, shell, git)
```

---

## Interactive REPL (`shellforge chat`)

Pair-programming mode. Persistent conversation history across prompts — the model remembers what you discussed.

```bash
shellforge chat                          # local model via Ollama (default)
shellforge chat --provider anthropic     # Anthropic API (Haiku/Sonnet/Opus)
shellforge chat --model qwen3:14b        # pick a specific model
```

Features:
- **Color output** — green prompt, red errors, yellow governance denials
- **Shell escapes** — `!git status` runs a command without leaving the session
- **Ctrl+C** — interrupts the current agent run without killing the session
- **Governance** — every tool call checked against `agentguard.yaml`, same as autonomous mode

---

## Ralph Loop (`shellforge ralph`)

Stateless-iterative multi-task execution. Each task gets a fresh context window — no accumulated confusion across tasks.

```bash
shellforge ralph tasks.json                    # run tasks from a JSON file
shellforge ralph --validate "go test ./..."    # validate after each task
shellforge ralph --dry-run                     # preview without executing
```

The loop: **PICK** a task → **IMPLEMENT** it → **VALIDATE** (run tests) → **COMMIT** on success → **RESET** context → next task.

Tasks come from a JSON file or Octi Pulpo MCP dispatch. Failed validations skip the commit and move on — no broken code lands.

---

## The Stack

| Layer | Project | What It Does |
|-------|---------|--------------|
| **Infer** | [Ollama](https://ollama.com) | Local LLM inference (Metal GPU on Mac) |
| **Optimize** | [RTK](https://github.com/rtk-ai/rtk) | Token compression — 70-90% reduction on shell output |
| **Execute** | [Goose](https://block.github.io/goose) | AI coding agent with native Ollama support (headless) |
| **Coordinate** | [Octi Pulpo](https://github.com/chitinhq/octi-pulpo) | Budget-aware dispatch, episodic memory, model cascading |
| **Govern** | [AgentGuard](https://github.com/chitinhq/agentguard) | Policy enforcement on every action — allow/deny/correct |
| **Sandbox** | [OpenShell](https://github.com/NVIDIA/OpenShell) | Kernel-level isolation (Docker on macOS) |
| **Scan** | [DefenseClaw](https://github.com/cisco-ai-defense/defenseclaw) | Supply chain scanner — AI Bill of Materials |

```bash
shellforge status
# Ollama        running (qwen3:30b loaded)
# RTK           v0.4.2
# AgentGuard    enforce mode (5 rules)
# Octi Pulpo    connected (http://localhost:8080)
# OpenShell     Docker sandbox active
# DefenseClaw   scanner ready
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `shellforge chat` | Interactive REPL — pair-program with a local or cloud model |
| `shellforge chat --provider anthropic` | REPL via Anthropic API (Haiku/Sonnet/Opus) |
| `shellforge chat --model qwen3:14b` | REPL with a specific Ollama model |
| `shellforge ralph tasks.json` | Multi-task loop — stateless-iterative execution |
| `shellforge ralph --validate "go test ./..."` | Ralph Loop with post-task validation |
| `shellforge ralph --dry-run` | Preview tasks without executing |
| `shellforge agent "prompt"` | One-shot governed agent (Ollama, default) |
| `shellforge agent --provider anthropic "prompt"` | One-shot via Anthropic API (prompt caching) |
| `shellforge agent --thinking-budget 8000 "prompt"` | Enable extended thinking (Sonnet/Opus) |
| `shellforge run <driver> "prompt"` | Run a governed CLI driver (goose, claude, copilot, codex, gemini) |
| `shellforge setup` | Install Ollama, create governance config, verify stack |
| `shellforge qa [dir]` | QA analysis — find test gaps and issues |
| `shellforge report [repo]` | Generate a status report from git + logs |
| `shellforge serve agents.yaml` | Daemon mode — run a 24/7 agent swarm |
| `shellforge status` | Show ecosystem health |
| `shellforge version` | Print version |

---

## Built-in Tools

The agent loop (used by `chat`, `agent`, and `ralph`) has 8 built-in tools, all governed:

| Tool | What It Does |
|------|-------------|
| `read_file` | Read file contents |
| `write_file` | Write a complete file |
| `edit_file` | Targeted find-and-replace (like Claude Code's Edit tool) |
| `glob` | Pattern-based file discovery with recursive `**` support |
| `grep` | Regex content search with `file:line` output |
| `run_shell` | Execute shell commands (via RTK for token compression) |
| `list_directory` | List directory contents |
| `search_files` | Search files by name pattern |

---

## Multi-Driver Governance

ShellForge governs any CLI agent driver via AgentGuard hooks. Each driver keeps its own model and agent loop — ShellForge ensures governance is active and spawns the driver as a subprocess.

```bash
# Run any driver with governance
shellforge run claude "review this code"
shellforge run codex "generate tests"
shellforge run copilot "update docs"
shellforge run gemini "security audit"
```

Orchestrate multiple drivers in a single Dagu DAG:

```bash
dagu start dags/multi-driver-swarm.yaml
```

See `dags/multi-driver-swarm.yaml` and `dags/workspace-swarm.yaml` for examples.

---

## Architecture

```
┌───────────────────────────────────────────────────┐
│  Entry Points                                      │
│  chat (REPL) · agent (one-shot) · ralph (multi)   │
│  run <driver> · serve (daemon)                     │
└────────────────────┬──────────────────────────────┘
                     │ prompt / task
┌────────────────────▼──────────────────────────────┐
│  Octi Pulpo (Coordination)                         │
│  Budget-aware dispatch · Memory · Model cascading  │
└────────────────────┬──────────────────────────────┘
                     │ task
┌────────────────────▼──────────────────────────────┐
│  ShellForge Agent Loop                             │
│  LLM provider · Tool calling · Drift detection     │
│  Sub-agent orchestrator (spawn sync/async)         │
│  Anthropic API or Ollama                           │
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
│  8 tools: read/write/edit/glob/grep/shell/ls/find  │
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

## The Governed Swarm Platform

| Project | Role | What It Does |
|---------|------|--------------|
| **ShellForge** | Orchestration | Governed agent runtime — CLI drivers + OpenClaw + local models |
| [**Octi Pulpo**](https://github.com/chitinhq/octi-pulpo) | Coordination | Swarm brain — shared memory, model routing, budget-aware dispatch |
| [**AgentGuard**](https://github.com/chitinhq/agentguard) | Governance | Policy enforcement, telemetry, invariants — on every tool call |
| [**AgentGuard Cloud**](https://github.com/chitinhq/agentguard-cloud) | Observability | SaaS dashboard — session replay, compliance, analytics |

ShellForge orchestrates. Octi Pulpo coordinates. AgentGuard governs.

### Supported Runtimes

| Runtime | What It Adds | Best For |
|---------|-------------|----------|
| **CLI Drivers** | Claude Code, Codex, Copilot, Gemini, Goose | Coding, PRs, commits |
| **[OpenClaw](https://github.com/openclaw/openclaw)** | Browser automation, 100+ skills, web app access | Integrations, NotebookLM, ChatGPT |
| **[NemoClaw](https://github.com/NVIDIA/NemoClaw)** | OpenClaw + NVIDIA OpenShell sandbox + Nemotron | Enterprise, air-gapped, zero-cost local inference |
| **[Ollama](https://ollama.com)** | Local model inference (Metal GPU) | Privacy, zero API cost |

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

**[Website](https://agentguardhq.github.io/shellforge)** · **[Star on GitHub](https://github.com/chitinhq/shellforge)** · **[AgentGuard](https://github.com/chitinhq/agentguard)**

*Built by humans and agents*

MIT License

</div>
