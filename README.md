# 🔥 ShellForge

A minimal, production-quality local agent swarm powered by [Ollama](https://ollama.com) and governed by [AgentGuard](https://github.com/AgentGuardHQ/agentguard).

Forge autonomous AI agents on your Mac (Apple Silicon) with local LLMs, full governance, and zero cloud dependency.

## Why

Cloud AI agents are powerful but expensive, rate-limited, and opaque. This repo gives you:

- **Local execution** — agents run on your machine via Ollama
- **Governance** — every agent action is policy-checked by AgentGuard
- **Low memory** — optimized for Apple Silicon, ~1.3 GB for a 1.7B model
- **Extensible** — plug in DeepAgents, OpenCode, or any framework later
- **Cron-ready** — every script is safe for repeated, unattended execution

## Quick Start

```bash
# 1. Clone
git clone https://github.com/AgentGuardHQ/shellforge.git
cd shellforge

# 2. Setup (installs deps, Ollama, pulls model)
bash scripts/setup.sh

# 3. Run an agent
npm run report                              # weekly status report
npm run qa                                  # analyze code for test gaps
npm run agent -- "build a CLI arg parser"   # prototype code
```

## Agents

| Agent | Input | Output | Use Case |
|-------|-------|--------|----------|
| **qa** | Source files | Test suggestions | Find untested code paths |
| **report** | Git log + agent logs | Markdown report | Weekly status summary |
| **prototype** | Prompt string | Code snippet | Quick scaffolding |

All agents are deterministic, bounded, and write output to `outputs/` only.

## Project Structure

```
shellforge/
├── agents/                 # Agent implementations
│   ├── qa-agent.ts         # Code analysis → test suggestions
│   ├── report-agent.ts     # Git + logs → markdown report
│   └── prototype-agent.ts  # Prompt → code snippet
├── config/
│   ├── ollama.ts           # Ollama HTTP wrapper
│   ├── agent-config.ts     # Agent definitions + routing
│   └── memory.ts           # Memory optimization (placeholder)
├── adapters/
│   ├── deepagents.ts       # DeepAgents integration (placeholder)
│   └── opencode.ts         # OpenCode integration (placeholder)
├── scripts/
│   ├── setup.sh            # One-time setup
│   ├── run-agent.sh        # Generic runner with governance
│   ├── run-qa-agent.sh     # Cron-safe QA wrapper
│   └── run-report-agent.sh # Cron-safe report wrapper
├── outputs/
│   ├── reports/            # Generated markdown reports
│   └── logs/               # Agent run logs
├── docs/
│   ├── architecture.md     # System design + diagrams
│   └── roadmap.md          # Feature roadmap
├── agentguard.yaml         # Governance policy
├── .env.example            # Configuration template
└── package.json
```

## AgentGuard Governance

The `agentguard.yaml` policy enforces:

| Policy | Action | Description |
|--------|--------|-------------|
| no-force-push | deny | Block `git push --force` |
| no-destructive-rm | deny | Block `rm -rf` |
| file-write-constraints | deny | Restrict writes to `outputs/` |
| test-before-merge | monitor | Log merge activity (stub) |
| bounded-execution | deny | 5-minute timeout per agent |

Default mode is **monitor** (log everything, block nothing). Switch to **enforce** when ready:

```yaml
mode: enforce  # in agentguard.yaml
```

## Configuration

Copy `.env.example` to `.env` and customize:

```bash
OLLAMA_MODEL=qwen3:1.7b     # Model to use (any Ollama-supported model)
OLLAMA_CTX_SIZE=4096         # Context window (lower = less RAM)
AGENT_TIMEOUT=300            # Max seconds per agent run
```

### Model Options

| Model | Size | RAM | Speed | Quality |
|-------|------|-----|-------|---------|
| qwen3:1.7b | 1.7B | ~1.2 GB | Fast | Good for simple tasks |
| qwen3:4b | 4B | ~3 GB | Medium | Better reasoning |
| mistral:7b | 7B | ~5 GB | Slower | Best quality |

## Cron Setup

Add to your crontab (`crontab -e`):

```cron
# Run QA agent every 10 minutes
*/10 * * * * /path/to/shellforge/scripts/run-qa-agent.sh

# Generate report every 30 minutes
*/30 * * * * /path/to/shellforge/scripts/run-report-agent.sh
```

Scripts are idempotent — safe for overlapping cron runs (timeout handles long runs).

## Extending with Frameworks

### DeepAgents

For multi-step autonomous planning:

```bash
npm install deepagents
```

Then implement `adapters/deepagents.ts` — the interface is already defined.
Wire it into `config/agent-config.ts` by adding `framework: 'deepagents'` to any agent config.

### OpenCode

For interactive coding with tool use:

```bash
npm install opencode
```

Then implement `adapters/opencode.ts`. AgentGuard hooks will automatically
govern OpenCode's tool calls through the existing policy layer.

See `docs/roadmap.md` for the full integration plan.

## Memory Optimization

`config/memory.ts` defines a pluggable memory layer with three functions:

- `initMemoryLayer()` — initialize the backend
- `optimizePrompt()` — compress/optimize prompts before sending to model
- `trackUsage()` — monitor token consumption

Currently stubbed (passthrough). When a memory library is ready (Google A2A, MemGPT, etc.),
swap the implementation — no other files need to change.

## Design Constraints

- **Max 2 concurrent agents** — Ollama serializes inference
- **No background daemons** — everything is script-based, exits cleanly
- **No distributed systems** — single machine only
- **Low memory priority** — default model fits in ~1.3 GB
- **Deterministic output** — low temperature, no random seeds

## How It Relates to AgentGuard

This repo is a **local companion** to the AgentGuard ecosystem:

- **[AgentGuard](https://github.com/AgentGuardHQ/agentguard)** — governance runtime (hooks, policies, telemetry)
- **[AgentGuard Cloud](https://github.com/AgentGuardHQ/agentguard-cloud)** — SaaS dashboard and analytics
- **ShellForge** (this repo) — local agent execution with governance

The local swarm can optionally send telemetry to AgentGuard Cloud for centralized observability.

## License

MIT
