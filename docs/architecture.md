# Architecture

## Overview

```
┌─────────────────────────────────────────────┐
│               Cron / Manual                 │
│         scripts/run-agent.sh <name>         │
└────────────────────┬────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│              AgentGuard Policy              │
│           agentguard.yaml (Layer 1)         │
│    allow/deny governance • audit logging    │
└────────────────────┬────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────┐
│               Agent (TypeScript)            │
│  qa-agent.ts │ report-agent.ts │ prototype  │
│  ─────────────────────────────────────────  │
│  input → prompt → model → output → save     │
└────────────────────┬────────────────────────┘
                     │
          ┌──────────┼──────────┐
          ▼          ▼          ▼
    ┌──────────┐ ┌────────┐ ┌────────────┐
    │  Ollama  │ │ Memory │ │  Adapters  │
    │  (local) │ │ Layer  │ │ (future)   │
    │  qwen3   │ │ (stub) │ │ deepagents │
    │  mistral │ │        │ │ opencode   │
    └──────────┘ └────────┘ └────────────┘
```

## Layers

### Layer 0 — Sandbox (future: NVIDIA OpenShell)
Kernel-level process isolation. Not implemented yet.
See: [research ticket](https://github.com/AgentGuardHQ/agentguard/issues/1036)

### Layer 1 — Governance (AgentGuard)
Policy-as-code enforcement via `agentguard.yaml`.
Currently in `monitor` mode — logs all actions, blocks nothing.
Switch to `enforce` when policies are battle-tested.

### Layer 2 — Agent Logic
Simple TypeScript scripts. Each agent:
1. Reads input (files, git log, user prompt)
2. Constructs a system + user prompt
3. Sends to Ollama via HTTP
4. Saves output to `outputs/`

No frameworks, no daemons, no state between runs.

### Layer 3 — Model (Ollama)
Local LLM serving via Ollama. Default model: `qwen3:1.7b` (1.7B params, ~1GB RAM).
Swap to any Ollama-supported model by changing `OLLAMA_MODEL` in `.env`.

## Data Flow

```
Input Sources          Agent               Output
─────────────          ─────               ──────
source files    ──→  qa-agent      ──→  outputs/logs/qa-*.log
git log + logs  ──→  report-agent  ──→  outputs/reports/report-*.md
user prompt     ──→  prototype     ──→  outputs/logs/prototype-*.log
```

## Memory Budget

| Component     | RAM     |
|---------------|---------|
| Ollama + 1.7B | ~1.2 GB |
| Node.js agent | ~50 MB  |
| **Total**     | **~1.3 GB** |

For larger models (7B): ~5 GB total. Apple Silicon unified memory makes this efficient.

## Concurrency

Max 2 concurrent agents assumed. Ollama serializes model inference,
so parallel agents queue at the model level. No explicit locking needed.
