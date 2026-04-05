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

### v0.6.0 — Goose + Governed Shell
- [x] Goose as local model driver (`shellforge run goose`)
- [x] `govern-shell.sh` — shell wrapper that evaluates every command through AgentGuard
- [x] `shellforge run goose` sets SHELL to governed wrapper automatically
- [x] Fixed catch-all deny bug (bounded-execution policy was denying everything)
- [x] Dagu DAG templates (sdlc-swarm, studio-swarm, workspace-swarm, multi-driver)

### v0.7.0 — Anthropic API Provider
- [x] LLM provider interface (`llm.Provider`) — pluggable Ollama vs Anthropic backends
- [x] Anthropic API adapter — stdlib HTTP, structured `tool_use` blocks, multi-turn history
- [x] Prompt caching — `cache_control: ephemeral` on system + tools, ~90% savings on cached tokens
- [x] Extended thinking budget (`--thinking-budget` flag)
- [x] Model cascading via Octi Pulpo (Haiku→Sonnet→Opus by `TaskComplexity` score)
- [x] Drift detection — self-score every 5 tool calls, steer below 7, kill below 5 twice
- [x] RTK token compression wired into `runShellWithRTK()` (70-90% savings on shell output)

### v0.8.0 — UMAAL (Interactive REPL + Ralph Loop + Enhanced Tools)
- [x] Interactive REPL (`shellforge chat`) — pair-programming with persistent conversation history
- [x] Color output (green prompt, red errors, yellow governance denials)
- [x] Shell escapes (`!command`) and Ctrl+C interrupt without session kill
- [x] Ollama (local) and Anthropic API provider support in REPL
- [x] Ralph Loop (`shellforge ralph`) — stateless-iterative multi-task execution
- [x] PICK → IMPLEMENT → VALIDATE → COMMIT → RESET cycle
- [x] Task input from JSON file or Octi Pulpo MCP dispatch
- [x] `--validate` flag for post-task test commands, `--dry-run` for preview
- [x] Sub-agent orchestrator — SpawnSync (block), SpawnAsync (fire and collect)
- [x] Concurrency control via semaphore, context compression (~750 tokens)
- [x] `edit_file` tool — targeted find-and-replace
- [x] `glob` tool — pattern-based file discovery with recursive `**` support
- [x] `grep` tool — regex content search with `file:line` output

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

### Phase 7.5 — Octi Pulpo Integration + Browser Drivers

ShellForge orchestrates, Octi Pulpo coordinates, AgentGuard governs. This phase wires the three together.

#### 7.5.1 — Octi Pulpo Coordination
- [ ] Consume Octi Pulpo MCP tools (route_recommend, coord_claim, coord_signal)
- [ ] Budget-aware driver selection — query Octi Pulpo before choosing model/driver
- [ ] Duplicate work prevention via coord_claim (prevents agent stampedes)
- [ ] Driver health signals — broadcast ShellForge agent status to Octi Pulpo

#### 7.5.2 — OpenClaw / NemoClaw Browser Driver
- [ ] OpenClaw as execution runtime for browser-based agents
- [ ] NemoClaw as optional adapter (never a dependency — protect kernel independence)
- [ ] Browser driver support in `shellforge run` (alongside Goose, Claude Code, Copilot, Codex, Gemini)
- [ ] Governed browser actions through AgentGuard kernel

#### 7.5.3 — Ecosystem Wiring
- [ ] ShellForge agents auto-connect to Octi Pulpo MCP server on startup
- [ ] Shared memory across ShellForge-managed agents via Octi Pulpo memory_store/recall
- [ ] Model routing delegation — ShellForge defers to Octi Pulpo route_recommend

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
- [ ] Publish Go module (`github.com/chitinhq/agentguard/go/pkg/hook`)
- [ ] Move `internal/` types to `pkg/` for external import
- [ ] Cloud telemetry opt-in (AgentGuard Cloud)

### Phase 11 — Replace Workspace Bash Swarm ✅ DONE
- [x] Migrated to API-driven dispatch: Octi Pulpo → ShellForge → Anthropic API
- [x] GH Actions Copilot Agent workflow (`dispatch-agent.yml`) for free-tier automation
- [x] ShellForge is now the execution harness for the agentguard-workspace swarm

---

## Bug Backlog (Open Issues)

Bugs identified during v0.6.x development. Fix before v1.0.

| Issue | Package | Severity | Description |
|-------|---------|----------|-------------|
| [#69](https://github.com/chitinhq/shellforge/issues/69) | `agentguard.yaml` | High | Governance gap: plain `rm` and `rm -r` bypass `no-destructive-rm` policy |
| [#67](https://github.com/chitinhq/shellforge/issues/67) | `scripts/govern-shell.sh` | Medium | Fragile `sed`-based JSON parsing — denial reason extraction can fail or corrupt |
| [#65](https://github.com/chitinhq/shellforge/issues/65) | `internal/scheduler` | Medium | `os.WriteFile` error silently ignored — audit log loss |
| [#63](https://github.com/chitinhq/shellforge/issues/63) | `internal/normalizer` | Medium | `classifyShellRisk` prefix match too broad — `catalog_tool` classified as read-only |
| [#62](https://github.com/chitinhq/shellforge/issues/62) | `cmd/shellforge` | Medium | `cmdEvaluate` ignores JSON unmarshal error — malformed input defaults to allow |
| [#61](https://github.com/chitinhq/shellforge/issues/61) | `internal/intent` | Low | Dead code in `flattenParams` — first assignment immediately overwritten |
| [#60](https://github.com/chitinhq/shellforge/issues/60) | all packages | High | Zero test coverage — critical for a governance runtime |

---

## Stack (as of v0.8.0)

| Component | Role | Status |
|---|---|---|
| `shellforge chat` | Interactive REPL | Working |
| `shellforge ralph` | Multi-task loop | Working |
| `shellforge agent` | One-shot agent | Working |
| Goose (Block) | Local model driver | Working |
| Claude Code | API driver (Linux) | Working (via hooks) |
| Copilot CLI | API driver (Linux) | Working (via hooks) |
| Codex CLI | API driver (Linux) | Coming soon |
| Gemini CLI | API driver (Linux) | Coming soon |
| Ollama | Local inference | Working |
| Anthropic API | Cloud inference | Working (prompt caching) |
| AgentGuard | Governance kernel | Working (YAML eval + Go kernel) |
| Octi Pulpo | Swarm coordination | Working (MCP) |
| RTK | Token compression | Optional |
| Docker | Sandbox | Optional |
