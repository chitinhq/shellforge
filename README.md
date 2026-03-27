<div align="center">

# 🔥 ShellForge

**Forge local AI agents. Governed. Private. Unstoppable.**

[![GitHub Pages](https://img.shields.io/badge/🌐_Live_Site-agentguardhq.github.io/shellforge-ff6b2b?style=for-the-badge)](https://agentguardhq.github.io/shellforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![AgentGuard](https://img.shields.io/badge/Governed_by-AgentGuard-green?style=for-the-badge)](https://github.com/AgentGuardHQ/agentguard)

*Run autonomous AI agents on your machine. No cloud. No API keys. No data leaves your laptop.*

[🌐 Website](https://agentguardhq.github.io/shellforge) · [📖 Docs](docs/architecture.md) · [🗺️ Roadmap](docs/roadmap.md) · [🛡️ AgentGuard](https://github.com/AgentGuardHQ/agentguard)

<img src="https://github.com/user-attachments/assets/a94a8a5e-dfeb-4771-a6ab-465d3c2f01f0" alt="ShellForge — Local Governed Agent Swarm" width="700">

</div>

---

## How It Works

Your agent thinks locally. AgentGuard sits **between every tool call and the outside world** — filesystem, shell, git, network. Nothing happens without policy approval.

```
┌──────────────────────────────────────────────────────────┐
│  Your Prompt                                             │
│  "Analyze this repo for test gaps"                       │
└──────────────────────┬───────────────────────────────────┘
                       │
              ┌────────▼────────┐
              │  ⚡ RTK          │  Strip 70-90% of terminal noise
              │  (token compress)│  before the LLM sees it
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │  🧠 TurboQuant   │  6x KV cache compression
              │  (memory optim)  │  run 14B models on 8GB Macs
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │  🦙 Ollama       │  Local inference
              │  qwen3 · mistral │  any GGUF model
              └────────┬────────┘
                       │
              ┌────────▼────────┐
              │  🔥 ShellForge   │  Agent execution loop
              │  agents/*.ts     │  input → prompt → model → action
              └────────┬────────┘
                       │
            ┌──────────▼──────────┐
            │  Agent wants to:     │
            │  write a file        │
            │  run a shell command │
            │  push to git         │
            │  fetch a URL         │
            └──────────┬──────────┘
                       │
         ══════════════╪══════════════
         ║  🛡️ AgentGuard            ║
         ║  Policy-as-code gateway   ║
         ║  allow / deny / audit     ║
         ║  every. single. action.   ║
         ══════════════╪══════════════
                       │
              ┌────────▼────────┐
              │  🌍 Environment  │  Filesystem · Shell · Git
              │  (the real world)│  Network · APIs
              └─────────────────┘
```

**AgentGuard is the gatekeeper.** The agent decides what to do. AgentGuard decides if it's allowed. Only approved actions reach your environment.

---

## The Ecosystem

Eight open-source projects. One governed agent runtime.

| Layer | Project | What It Does |
|-------|---------|--------------|
| ⚡ **Optimize** | [RTK](https://github.com/rtk-ai/rtk) | Rust Token Killer — 70-90% fewer tokens to the LLM |
| 🧠 **Compress** | TurboQuant (Google, ICLR 2026) | 3-bit KV cache — 6x memory reduction, zero accuracy loss |
| 🦙 **Infer** | [Ollama](https://ollama.com) | Local model serving — any GGUF on your Mac |
| 🔥 **Execute** | **ShellForge** ← you are here | Agent runtime — TypeScript scripts, zero frameworks |
| 🛡️ **Govern** | [AgentGuard](https://github.com/AgentGuardHQ/agentguard) | Policy gateway between tool calls and your environment |
| 🐾 **Scan** | [DefenseClaw](https://github.com/cisco-ai-defense/defenseclaw) (Cisco) | Supply chain — scan agent skills + MCP servers |
| 🔒 **Sandbox** | [OpenShell](https://github.com/NVIDIA/OpenShell) (NVIDIA) | Kernel isolation — Landlock + Seccomp BPF |
| 🤖 **Plan** | [DeepAgents](https://github.com/langchain-ai/deepagents) (LangChain) | Multi-step autonomous planning + tool use |

> **Coming soon:** Native integrations with RTK, TurboQuant, OpenShell, DefenseClaw, and DeepAgents. [Star this repo](https://github.com/AgentGuardHQ/shellforge) to follow along.

---

## Quick Start

```bash
git clone https://github.com/AgentGuardHQ/shellforge.git
cd shellforge
bash scripts/setup.sh       # installs deps, Ollama, pulls model

npm run report               # generate a weekly status report
npm run qa                   # analyze code for test gaps
npm run agent -- "prompt"    # prototype code from a prompt
```

**Requirements:** macOS (Apple Silicon) or Linux · Node 20+ · ~1.3 GB RAM

---

## What It Does

ShellForge runs **local AI agents** powered by [Ollama](https://ollama.com) with full [AgentGuard](https://github.com/AgentGuardHQ/agentguard) governance. Each agent is a bounded TypeScript script — no frameworks, no daemons, no complexity.

### Built-in Agents

| Agent | Command | Input → Output |
|-------|---------|----------------|
| 🔍 **QA** | `npm run qa` | Source files → test suggestions |
| 📊 **Report** | `npm run report` | Git log + activity → markdown summary |
| ⚡ **Prototype** | `npm run agent -- "prompt"` | Prompt → code snippet |

### Governance Policies

```yaml
# agentguard.yaml — every agent runs under these rules
mode: monitor  # → switch to 'enforce' when battle-tested

policies:
  - no-force-push      # block git push --force
  - no-destructive-rm  # block rm -rf
  - file-write-bounds  # agents can only write to outputs/
  - bounded-execution  # 5-minute timeout per run
```

---

## Architecture

```
┌─────────────────────────────────┐
│  🤖 Agent (TypeScript)          │  ← agents/*.ts
│  input → prompt → model → action│
└───────────────┬─────────────────┘
                │ tool call
    ════════════╪════════════
    ║ 🛡️ AgentGuard          ║  ← agentguard.yaml
    ║ allow/deny · audit      ║     the gatekeeper
    ════════════╪════════════
                │ approved action
        ┌───────┼───────┐
        ▼       ▼       ▼
    ┌────────┐ ┌─────┐ ┌──────────┐
    │ Ollama │ │ FS  │ │ Shell/   │
    │ qwen3  │ │ Git │ │ Network  │
    └────────┘ └─────┘ └──────────┘
         🌍 Your Environment
```

**Memory budget:** ~1.3 GB (1.7B model) or ~5 GB (7B model). Apple Silicon unified memory makes this efficient.

See [docs/architecture.md](docs/architecture.md) for the full design.

---

## Configuration

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `qwen3:1.7b` | Any Ollama-supported model |
| `OLLAMA_CTX_SIZE` | `4096` | Context window (lower = less RAM) |
| `AGENT_TIMEOUT` | `300` | Max seconds per agent run |
| `AGENT_OUTPUT_DIR` | `outputs` | Directory for agent output files |

### Model Options

| Model | Params | RAM | Best For |
|-------|--------|-----|----------|
| `qwen3:1.7b` | 1.7B | ~1.2 GB | Fast tasks, prototyping |
| `qwen3:4b` | 4B | ~3 GB | Balanced reasoning |
| `mistral:7b` | 7B | ~5 GB | Complex analysis |

---

## Cron Automation

```cron
# crontab -e
*/10 * * * * /path/to/shellforge/scripts/run-qa-agent.sh
*/30 * * * * /path/to/shellforge/scripts/run-report-agent.sh
```

All scripts are idempotent and timeout-safe.

---

## Extensibility

ShellForge is designed to grow. Adapter interfaces are already defined — swap in real implementations when ready.

### Framework Adapters (Planned)

| Framework | Adapter | Status |
|-----------|---------|--------|
| [DeepAgents](https://github.com/deepagents) | `adapters/deepagents.ts` | 🔌 Interface ready |
| [OpenCode](https://github.com/opencode) | `adapters/opencode.ts` | 🔌 Interface ready |
| NVIDIA OpenShell | — | 🔬 [Research](https://github.com/AgentGuardHQ/agentguard/issues/1036) |
| Cisco DefenseClaw | — | 🔬 [Research](https://github.com/AgentGuardHQ/agentguard/issues/1036) |

### Memory Optimization (Planned)

`config/memory.ts` exposes a pluggable interface:

```typescript
initMemoryLayer()   // initialize backend (Google A2A, MemGPT, custom)
optimizePrompt()    // compress context before model call
trackUsage()        // monitor token consumption
```

Currently passthrough — swap in a real backend with zero refactor.

---

## The Ecosystem

ShellForge is part of the **AgentGuard** platform:

| Project | What It Does |
|---------|--------------|
| [**AgentGuard**](https://github.com/AgentGuardHQ/agentguard) | Governance gateway — sits between every agent tool call and your environment. Hooks for Claude Code, Codex, Copilot, Gemini, OpenCode, DeepAgents. |
| [**AgentGuard Cloud**](https://github.com/AgentGuardHQ/agentguard-cloud) | SaaS dashboard — observability, session replay, swarm org chart |
| **ShellForge** | Local agent runtime — Ollama + governance on your machine |
| [**RTK**](https://github.com/rtk-ai/rtk) | Rust Token Killer — compress terminal output 70-90% before the LLM |
| **TurboQuant** | Google KV cache compression — 6x memory reduction for local models |
| [**DefenseClaw**](https://github.com/cisco-ai-defense/defenseclaw) | Cisco supply chain security — scan agent skills + MCP servers |
| [**OpenShell**](https://github.com/NVIDIA/OpenShell) | NVIDIA kernel sandbox — Landlock + Seccomp process isolation |
| [**DeepAgents**](https://github.com/langchain-ai/deepagents) | LangChain multi-step planning — autonomous task decomposition |

---

## Design Philosophy

- **No daemons.** Every agent is a script that runs and exits.
- **No frameworks.** Raw TypeScript + Ollama HTTP. Add frameworks when you need them.
- **No cloud required.** Everything runs locally. Cloud telemetry is opt-in.
- **No magic.** Read any agent in 60 seconds. Fork and customize in 5 minutes.

---

## Contributing

ShellForge is open source and early. We welcome:

- New agent implementations
- Framework adapter integrations
- Memory optimization backends
- Governance policy templates

See [docs/roadmap.md](docs/roadmap.md) for what's planned.

---

<div align="center">

**[🌐 Website](https://agentguardhq.github.io/shellforge)** · **[⭐ Star on GitHub](https://github.com/AgentGuardHQ/shellforge)** · **[🛡️ AgentGuard](https://github.com/AgentGuardHQ/agentguard)**

*Built with 🔥 by humans and agents*

MIT License

</div>
