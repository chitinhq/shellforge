# ShellForge Roadmap

## Completed

### v0.1.0 — Foundation
- [x] Go binary with Ollama integration
- [x] 3 agents (QA, report, prototype)
- [x] agentguard.yaml governance (enforce/monitor)
- [x] Cron-based scheduling

### v0.2.0 — Release Pipeline
- [x] Goreleaser + Homebrew tap (`brew install shellforge`)
- [x] GitHub Pages site
- [x] `shellforge serve` — daemon mode with memory-aware scheduling
- [x] Terminal Bench 2.0 Harbor adapter

### v0.3.x — Multi-Driver
- [x] `shellforge run <driver>` — launch governed agents
- [x] Driver support: Claude Code, Copilot CLI, Codex, Gemini
- [x] Format-agnostic intent parser (extracts tool calls from any model output)
- [x] Normalizer (raw tool call → Canonical Action Representation)
- [x] Correction engine (denial → feedback → retry)
- [x] Setup wizard (6-step interactive installer)

### v0.4.x — Environment Awareness
- [x] Server mode (Linux, no GPU) — skips Ollama, shows API drivers
- [x] Mac mode — local models via Ollama
- [x] `shellforge evaluate` — JSON governance evaluation endpoint
- [x] `shellforge swarm` — starts Dagu orchestration dashboard

### v0.5.x — Driver Iteration
- [x] Tested Crush (broken — OpenAI-compat shim loses tool calls)
- [x] Tested Aider (file editing only, no shell execution)
- [x] Evaluated Goose (Block) — native Ollama, actually executes tools

### v0.6.0 — Goose + Governed Shell ← CURRENT
- [x] Goose as local model driver (`shellforge run goose`)
- [x] `govern-shell.sh` — shell wrapper that evaluates every command through AgentGuard
- [x] `shellforge run goose` sets SHELL to governed wrapper automatically
- [x] Fixed catch-all deny bug (bounded-execution policy was denying everything)
- [x] Dagu DAG templates (sdlc-swarm, studio-swarm, workspace-swarm, multi-driver)

---

## In Progress

### Phase 7 — Governed Multi-Agent Architecture
Foundation types exist (`internal/action/`, `internal/orchestrator/`, `internal/scheduler/queue.go`) but not wired into execution.

#### 7.1 — Wire Orchestrator
- [ ] Connect orchestrator state machine to `shellforge run`
- [ ] Proposal → Governance → Result flow through kernel
- [ ] Run-level audit trail (structured events, not just logs)

#### 7.2 — Turn-Based Swarm
- [ ] Planner agent — task decomposition via Ollama
- [ ] Worker agent — Goose executes subtasks with governance
- [ ] Evaluator agent — validates results
- [ ] State machine: PLANNING → WORKING → EVALUATING → COMPLETE

#### 7.3 — Resilience
- [ ] Anti-loop hash detection
- [ ] Escalation thresholds (auto-fail after N denials)
- [ ] Circuit breaker on Ollama failures

#### 7.4 — Observability
- [ ] Structured event emission to SQLite
- [ ] Run summaries with governance stats
- [ ] 24h soak test

---

## Planned

### Phase 8 — AgentGuard MCP Server
- [ ] MCP server exposing governed tools
- [ ] Goose → MCP → AgentGuard → execute
- [ ] Dual-layer: kernel enforces, MCP integrates

### Phase 9 — Terminal Bench 2.0
- [x] Harbor adapter
- [ ] Dry run on single task with Goose
- [ ] Full 89-task evaluation
- [ ] Leaderboard submission

### Phase 10 — Production Hardening
- [ ] AgentGuard Go kernel integration (in-process, not subprocess)
- [ ] Publish Go module (`github.com/AgentGuardHQ/agentguard/go/pkg/hook`)
- [ ] Move `internal/` types to `pkg/` for external import
- [ ] Cloud telemetry opt-in (AgentGuard Cloud)

### Phase 11 — Replace Workspace Bash Swarm
- [ ] Dagu replaces `server/deploy.sh` + cron + queue.txt
- [ ] Multi-driver DAGs: Claude Code + Copilot + Codex on Linux box
- [ ] Same governance policy across all drivers
- [ ] ShellForge as the runtime for agentguard-workspace swarm

---

## Bug Backlog (Open Issues)

Bugs identified during v0.6.x development. Fix before v1.0.

| Issue | Package | Severity | Description |
|-------|---------|----------|-------------|
| [#69](https://github.com/AgentGuardHQ/shellforge/issues/69) | `agentguard.yaml` | High | Governance gap: plain `rm` and `rm -r` bypass `no-destructive-rm` policy |
| [#67](https://github.com/AgentGuardHQ/shellforge/issues/67) | `scripts/govern-shell.sh` | Medium | Fragile `sed`-based JSON parsing — denial reason extraction can fail or corrupt |
| [#65](https://github.com/AgentGuardHQ/shellforge/issues/65) | `internal/scheduler` | Medium | `os.WriteFile` error silently ignored — audit log loss |
| [#63](https://github.com/AgentGuardHQ/shellforge/issues/63) | `internal/normalizer` | Medium | `classifyShellRisk` prefix match too broad — `catalog_tool` classified as read-only |
| [#62](https://github.com/AgentGuardHQ/shellforge/issues/62) | `cmd/shellforge` | Medium | `cmdEvaluate` ignores JSON unmarshal error — malformed input defaults to allow |
| [#61](https://github.com/AgentGuardHQ/shellforge/issues/61) | `internal/intent` | Low | Dead code in `flattenParams` — first assignment immediately overwritten |
| [#60](https://github.com/AgentGuardHQ/shellforge/issues/60) | all packages | High | Zero test coverage — critical for a governance runtime |

---

## Stack (as of v0.6.1)

| Component | Role | Status |
|---|---|---|
| Goose (Block) | Local model driver | Working |
| Claude Code | API driver (Linux) | Working (via hooks) |
| Copilot CLI | API driver (Linux) | Working (via hooks) |
| Codex CLI | API driver (Linux) | Coming soon |
| Gemini CLI | API driver (Linux) | Coming soon |
| Ollama | Local inference | Working |
| AgentGuard | Governance kernel | Working (YAML eval + Go kernel) |
| Dagu | Orchestration | Working (DAGs + web UI) |
| RTK | Token compression | Optional |
| Docker | Sandbox | Optional |
