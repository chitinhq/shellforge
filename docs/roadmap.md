# Roadmap

## Phase 1 — Foundation ✅
- [x] Ollama integration with low-context wrapper
- [x] 3 simple agents (QA, report, prototype)
- [x] AgentGuard governance policy (monitor mode)
- [x] Script-based execution with cron support
- [x] Memory optimization placeholder

## Phase 2 — Hardening ✅
- [x] Go rewrite — single static binary (~7.5MB), zero Node.js dependencies
- [x] Switch agentguard.yaml to `enforce` mode
- [x] AgentGuard CLI hooks integrated into governance engine
- [x] Token budget tracking per agent per day
- [x] Output quality scoring (simple heuristics)
- [x] Error recovery and retry logic

## Phase 3 — Framework Integration ✅
- [x] **OpenCode** — Go CLI AI coding framework
  - Pluggable engine interface (`internal/engine/`)
  - `--non-interactive` subprocess mode, governance-wrapped
  - Tool-use governance via AgentGuard policy engine
- [x] **DeepAgents** — multi-agent orchestration (LangChain-based)
  - Subprocess engine adapter (`internal/engine/`)
  - Agent decomposition: break goals into sub-tasks
  - Governance-wrapped tool calls

## Phase 4 — Memory & Context ✅
- [x] **RTK v0.31.0** — token compression integrated
  - Auto-wraps shell output and LLM I/O
  - Reduces context window usage by ~40%
- [x] **TurboQuant** — model quantization + KV cache optimization
  - PyTorch MPS backend on Apple Silicon
  - Integrated via `internal/integration/`
- [x] Prompt caching for repeated patterns

## Phase 5 — Security ✅
- [x] **NVIDIA OpenShell** sandbox integration
  - Landlock + Seccomp isolation per agent run
  - Docker/Colima on Mac for Linux kernel features
  - Integrated via `internal/integration/`
- [x] **Cisco DefenseClaw** scanning
  - AI Bill of Materials (BoM) scanner
  - Scan agent skills/plugins pre-install
  - Integrated via `internal/integration/`

## Phase 6 — Scale ✅
- [x] Interactive setup CLI (`shellforge setup`)
- [x] Ecosystem health check (`shellforge status`)
- [x] Binary releases via goreleaser + Homebrew tap (#22)
- [x] `shellforge serve` — daemon mode with memory-aware scheduling (#32)
- [x] Paperclip as 9th integration (#31)
- [x] Terminal Bench 2.0 adapter for Harbor framework (#34)
- [x] Site updates — brew install CTA + swarm mode docs (#30, #33)
- [ ] Multi-model routing (qwen for fast, mistral for quality)
- [ ] Cross-platform support (Linux arm64, Windows)
- [ ] Cloud telemetry integration (AgentGuard Cloud)
- [ ] Dashboard for local swarm observability

## Phase 7 — Governed Multi-Agent Architecture 🔄 In Progress

Production-grade local-first multi-agent orchestration with governance at every boundary.

### Phase 7.1 — Foundation 🔄 In Progress
- [x] `ActionProposal` and `ActionResult` core types (`internal/action/types.go`)
- [x] `InferenceQueue` with semaphore-based concurrency (`internal/scheduler/queue.go`)
- [x] Orchestrator state machine with valid transitions (`internal/orchestrator/state.go`)
- [ ] Single-agent orchestrator with governance boundary
- [ ] Wire orchestrator into `shellforge run`

### Phase 7.2 — Turn-Based Swarm
- [ ] Planner agent — decomposes task into `ActionProposal` sequence
- [ ] Worker agent — executes proposals through governance gate
- [ ] Evaluator agent — scores results and decides next step
- [ ] Explicit state machine: PLANNING → WORKING → EVALUATING → COMPLETE
- [ ] Run-level context sharing between agents

### Phase 7.3 — Correction + Resilience
- [ ] Corrector role — rewrites denied proposals within policy
- [ ] Denial tracking with per-run and per-agent counters
- [ ] Anti-loop hash detection (repeated proposal fingerprints)
- [ ] Escalation thresholds — auto-fail after N consecutive denials
- [ ] Circuit breaker — pause swarm on systemic governance failures

### Phase 7.4 — Observability + Production
- [ ] Full event emission (proposal, decision, result, transition)
- [ ] Run summaries with governance statistics
- [ ] Terminal Bench 2.0 integration for multi-agent evaluation
- [ ] 24h soak test — sustained swarm stability under load

## Phase 8 — Terminal Bench 2.0 Submission
- [x] Harbor adapter (#34)
- [ ] Dry run on single task
- [ ] Full 89-task evaluation
- [ ] Leaderboard submission
