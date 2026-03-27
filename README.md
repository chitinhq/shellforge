<div align="center">

# 🔥 ShellForge

**Forge local AI agents. Governed. Private. Unstoppable.**

[![GitHub Pages](https://img.shields.io/badge/🌐_Live_Site-agentguardhq.github.io/shellforge-ff6b2b?style=for-the-badge)](https://agentguardhq.github.io/shellforge)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![AgentGuard](https://img.shields.io/badge/Governed_by-AgentGuard-green?style=for-the-badge)](https://github.com/AgentGuardHQ/agentguard)

*Run autonomous AI agents on your machine. No cloud. No API keys. No data leaves your laptop.*

[🌐 Website](https://agentguardhq.github.io/shellforge) · [📖 Docs](docs/architecture.md) · [🗺️ Roadmap](docs/roadmap.md) · [🛡️ AgentGuard](https://github.com/AgentGuardHQ/agentguard)

</div>

---

## The Vision

The agentic AI security stack is forming:

| Layer | Project | Role |
|-------|---------|------|
| **Layer 0** | [NVIDIA OpenShell](https://github.com/NVIDIA/OpenShell) | Kernel sandbox — Landlock + Seccomp isolation |
| **Layer 1** | [AgentGuard](https://github.com/AgentGuardHQ/agentguard) | Policy engine — allow/deny governance hooks |
| **Layer 2** | [Cisco DefenseClaw](https://github.com/cisco-ai-defense/defenseclaw) | Supply chain — skill scanning + MCP verification |
| **Layer 3** | **ShellForge** ← you are here | Agent runtime — local execution with full governance |

ShellForge is where the agents actually run. The other layers make sure they behave.

> **Coming soon:** Native integrations with OpenShell (sandbox), DefenseClaw (scanning), and AgentGuard Cloud (observability). [Star this repo](https://github.com/AgentGuardHQ/shellforge) to follow along.

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
Cron / CLI
    │
    ▼
┌─────────────────────────────────┐
│  🛡️ AgentGuard Policy Layer     │  ← agentguard.yaml
│  allow/deny · audit · telemetry │
└───────────────┬─────────────────┘
                │
                ▼
┌─────────────────────────────────┐
│  🤖 Agent (TypeScript)          │  ← agents/*.ts
│  input → prompt → model → save  │
└───────────────┬─────────────────┘
                │
        ┌───────┼───────┐
        ▼       ▼       ▼
    ┌────────┐ ┌─────┐ ┌──────────┐
    │ Ollama │ │ Mem │ │ Adapters │
    │ qwen3  │ │ (…) │ │ (future) │
    └────────┘ └─────┘ └──────────┘
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
| `OLLAMA_MODEL` | `qwen3:1.7b` | Any Ollama-supported model |
| `OLLAMA_CTX_SIZE` | `4096` | Context window (lower = less RAM) |
| `AGENT_TIMEOUT` | `300` | Max seconds per agent run |

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
| [**AgentGuard**](https://github.com/AgentGuardHQ/agentguard) | Governance runtime — hooks, policies, telemetry for Claude, Codex, Copilot, Gemini |
| [**AgentGuard Cloud**](https://github.com/AgentGuardHQ/agentguard-cloud) | SaaS dashboard — observability, session replay, swarm org chart |
| **ShellForge** | Local agent execution — Ollama + governance on your machine |

Combined with the emerging open-source security stack (**OpenShell** for sandboxing, **DefenseClaw** for supply chain scanning), this creates a full-stack governed agentic AI platform — from kernel isolation to cloud observability.

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
