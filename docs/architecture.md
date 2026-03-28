# ShellForge Architecture

## Overview

ShellForge is a single Go binary (~7.5MB) that provides governed local AI agent execution. Its core value is **governance** — when frameworks like OpenCode or DeepAgents are installed, they provide the agentic loop; ShellForge wraps them with AgentGuard policy enforcement.

## 8-Layer Stack

```
┌─────────────────────────────────────────────┐
│  Layer 8: OpenShell (Kernel Sandbox)        │  Docker/Colima isolation
├─────────────────────────────────────────────┤
│  Layer 7: DefenseClaw (Supply Chain)        │  Cisco AI BoM Scanner
├─────────────────────────────────────────────┤
│  Layer 6: Dagu (Orchestration)              │  YAML DAG workflows + web UI
├─────────────────────────────────────────────┤
│  Layer 5: Goose / OpenCode (Execution)      │  Primary local agent driver
├─────────────────────────────────────────────┤
│  Layer 4: AgentGuard (Governance Kernel)    │  Policy enforcement
├─────────────────────────────────────────────┤
│  Layer 3: TurboQuant (Quantization)         │  KV cache optimization (optional)
├─────────────────────────────────────────────┤
│  Layer 2: RTK (Token Compression)           │  Auto-compress I/O (optional)
├─────────────────────────────────────────────┤
│  Layer 1: Ollama (Local LLM)                │  Metal GPU on Mac
└─────────────────────────────────────────────┘
```

## Go Project Layout

```
cmd/shellforge/
├── main.go         # CLI entry point (cobra-style subcommands)
└── status.go       # Ecosystem health check

internal/
├── governance/     # agentguard.yaml parser + policy engine
├── ollama/         # Ollama HTTP client (chat, generate)
├── agent/          # Native fallback agentic loop
├── tools/          # 5 tool implementations + RTK wrapper
├── engine/         # Pluggable engine interface (OpenCode, DeepAgents)
├── logger/         # Structured JSON logging
└── integration/    # RTK, OpenShell, DefenseClaw, TurboQuant, AgentGuard
```

## Engine Architecture

ShellForge uses a pluggable engine system:

1. **Goose (Block)** (preferred local driver) — subprocess, native Ollama support, SHELL wrapped via `govern-shell.sh`
2. **OpenCode** (alternative) — subprocess, `--non-interactive` mode, governance-wrapped
3. **DeepAgents** (alternative) — subprocess, Node.js/Python SDK, governance-wrapped
4. **Native** (fallback) — built-in multi-turn loop with Ollama + tool calling

The engine selection is automatic based on what's installed. Use `shellforge run goose` for local models, or `shellforge agent` for the native loop.

## Governance Flow

```
User Request → Engine (Goose/OpenCode/DeepAgents/Native)
  → Tool Call → Governance Check (agentguard.yaml)
    → ALLOW → Execute Tool → Return Result
    → DENY  → Log Violation → Correction Feedback → Retry
```

## Data Flow

1. User invokes `./shellforge qa` (or agent, report, scan)
2. CLI loads `agentguard.yaml` governance policy
3. Detects available engine (Goose > OpenCode > DeepAgents > Native)
4. Engine sends prompt to Ollama (via RTK for token compression)
5. LLM responds with tool calls
6. Each tool call passes through governance check
7. Allowed tools execute (shell commands wrapped by RTK + OpenShell sandbox)
8. Results compressed by RTK, fed back to LLM
9. Loop continues until task complete or budget exhausted

## macOS (Apple Silicon) Support

All 8 layers run on Mac M4:
- Ollama uses Metal for GPU acceleration
- RTK, AgentGuard, OpenCode are native arm64 binaries
- TurboQuant runs via PyTorch (MPS backend)
- OpenShell runs inside Docker/Colima (Linux VM for Landlock)
- DefenseClaw installs via pip or source build
