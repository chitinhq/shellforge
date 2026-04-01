# ShellForge Architecture

## Overview

ShellForge is a single Go binary (~7.5MB) that provides governed AI agent execution. Its core value is **governance** — every agent driver, whether a CLI tool, browser session, or local model, runs through AgentGuard policy enforcement on every action.

## Execution Model

ShellForge supports three classes of agent driver, all governed uniformly:

```
┌─────────────────────────────────────────────────────────────┐
│  CLI Drivers (coding)                                       │
│  Claude Code · Codex · Copilot CLI · Gemini CLI · Goose    │
├─────────────────────────────────────────────────────────────┤
│  OpenClaw / NemoClaw (browser + integrations)               │
│  Web apps · NotebookLM · ChatGPT · Slack · 100+ skills     │
├─────────────────────────────────────────────────────────────┤
│  Local Models (zero cost)                                   │
│  Ollama · Nemotron (via NemoClaw)                           │
└─────────────────────────────────────────────────────────────┘
         │ every tool call
    ═════╪═════════════════
    ║  AgentGuard Kernel  ║
    ║  allow · deny · audit║
    ═════╪═════════════════
         │ approved
    Octi Pulpo (coordination)
    ─────┼─────────────────
         │
    Your Environment (files, shell, git, browser, APIs)
```

### CLI Drivers

Purpose-built for code generation. Each uses its own subscription — no API keys needed:

| Driver | Subscription | Best For |
|--------|-------------|----------|
| `claude-code` | Claude Max | Complex reasoning, architecture |
| `codex` | OpenAI Pro | Code generation, refactoring |
| `copilot` | GitHub Pro | PR workflows, code review |
| `gemini-cli` | Google AI Premium | Analysis, multi-modal |
| `goose` | Free (local Ollama) | Air-gapped, zero cost |

### OpenClaw / NemoClaw Runtime

Browser automation and integrations via consumer app subscriptions:

| App | Via | Capability |
|-----|-----|-----------|
| ChatGPT | Browser (Playwright) | Reasoning tasks via existing OpenAI Plus subscription |
| NotebookLM | Browser (Playwright) | Audio briefings, slide decks, charts, Drive docs |
| Gemini App | Browser (Playwright) | Multi-modal analysis via Google AI Premium |
| Slack, Discord | OpenClaw skills | Messaging, notifications, integrations |

**NemoClaw** (optional, heavier) adds:
- **NVIDIA OpenShell** — kernel-level sandbox (process isolation, not just policy)
- **Nemotron** — local NVIDIA models for zero-cost inference

### Local Models

Zero token cost via Ollama or Nemotron:

| Model | Params | RAM | Best For |
|-------|--------|-----|----------|
| `qwen3:1.7b` | 1.7B | ~1.2 GB | Fast triage, classification |
| `qwen3:8b` | 8B | ~6 GB | Balanced reasoning |
| `qwen3:30b` | 30B | ~19 GB | Production quality |
| Nemotron (via NemoClaw) | Various | GPU | NVIDIA hardware acceleration |

## The Governed Swarm Platform

ShellForge is one layer in a three-part platform:

```
┌─────────────────────────────────────────────────────┐
│  ShellForge                                         │
│  Orchestration — forge and run agent swarms         │
│  CLI drivers + OpenClaw/NemoClaw + local models     │
├─────────────────────────────────────────────────────┤
│  Octi Pulpo                                         │
│  Coordination — shared memory, model routing,       │
│  budget-aware dispatch, priority signaling           │
├─────────────────────────────────────────────────────┤
│  AgentGuard                                         │
│  Governance — policy enforcement, telemetry,         │
│  invariants, compliance                              │
└─────────────────────────────────────────────────────┘
```

ShellForge orchestrates. Octi Pulpo coordinates. AgentGuard governs.

## Cost-Aware Routing

Octi Pulpo routes tasks to the cheapest capable driver:

| Tier | Driver | Cost | Use When |
|------|--------|------|----------|
| Local | Ollama / Nemotron | $0 | Simple tasks, triage, classification |
| Subscription | Browser → ChatGPT / NotebookLM / Gemini | Already paying | Medium tasks, artifacts, briefings |
| CLI | Claude Code / Codex / Copilot | Already paying | Coding, PRs, commits |
| API | Direct API calls | Per-token | Burst capacity, programmatic access |

## Infrastructure Stack

| Layer | Project | What It Does |
|-------|---------|--------------|
| **Infer** | [Ollama](https://ollama.com) | Local LLM inference (Metal GPU on Mac) |
| **Optimize** | [RTK](https://github.com/rtk-ai/rtk) | Token compression — 70-90% reduction on shell output |
| **Execute** | [Goose](https://block.github.io/goose) / [OpenClaw](https://github.com/openclaw/openclaw) | Agent execution + browser automation |
| **Orchestrate** | [Dagu](https://github.com/dagu-org/dagu) | YAML DAG workflows with scheduling and web UI |
| **Coordinate** | [Octi Pulpo](https://github.com/AgentGuardHQ/octi-pulpo) | Swarm coordination via MCP |
| **Govern** | [AgentGuard](https://github.com/AgentGuardHQ/agentguard) | Policy enforcement on every action |
| **Sandbox** | [OpenShell](https://github.com/NVIDIA/OpenShell) | Kernel-level isolation (Docker on macOS) |
| **Scan** | [DefenseClaw](https://github.com/cisco-ai-defense/defenseclaw) | Supply chain scanner — AI Bill of Materials |

## Go Project Layout

```
cmd/shellforge/
├── main.go         # CLI entry point (cobra-style subcommands)
└── status.go       # Ecosystem health check

internal/
├── llm/            # LLM provider interface
│   ├── provider.go # Provider interface (Chat, Name) + Message/Response types
│   └── anthropic.go# Anthropic API adapter (stdlib HTTP, prompt caching, tool_use)
├── agent/          # Agentic loop
│   ├── loop.go     # runProviderLoop (Anthropic) + runOllamaLoop, drift detection wiring
│   └── drift.go    # Drift detector — self-score every 5 calls, steer/kill on low scores
├── governance/     # agentguard.yaml parser + policy engine
├── ollama/         # Ollama HTTP client (chat, generate)
├── tools/          # 5 tool implementations + RTK wrapper
├── engine/         # Pluggable engine interface (Goose, OpenClaw, OpenCode)
├── logger/         # Structured JSON logging
├── scheduler/      # Memory-aware scheduling + cron
├── orchestrator/   # Multi-agent state machine
├── normalizer/     # Canonical Action Representation
├── correction/     # Denial tracking + escalation
├── intent/         # Format-agnostic intent parsing
└── integration/    # RTK, OpenShell, DefenseClaw, TurboQuant, AgentGuard
```

## Engine Architecture

ShellForge uses a pluggable engine system:

1. **Goose** (preferred local driver) — subprocess, native Ollama support, SHELL wrapped via `govern-shell.sh`
2. **OpenClaw** (browser + integrations) — browser automation, web app access, 100+ skills
3. **NemoClaw** (enterprise) — OpenClaw + NVIDIA OpenShell sandbox + Nemotron local models
4. **CLI Drivers** (cloud coding) — Claude Code, Codex, Copilot CLI, Gemini CLI
5. **Native** (fallback) — built-in multi-turn loop with Ollama + tool calling

## Governance Flow

```
User Request → Engine (Goose/OpenClaw/CLI/Native)
  → Tool Call → Governance Check (agentguard.yaml)
    → ALLOW → Execute Tool → Return Result
    → DENY  → Log Violation → Correction Feedback → Retry
```

The format-agnostic intent parser handles tool calls from any LLM output format (tool_calls, JSON blocks, XML tags, function_call).

## macOS (Apple Silicon) Support

All layers run on Mac M4:
- Ollama uses Metal for GPU acceleration
- RTK, AgentGuard, ShellForge are native arm64 binaries
- OpenShell runs inside Docker/Colima (Linux VM for Landlock)
- DefenseClaw installs via pip or source build
