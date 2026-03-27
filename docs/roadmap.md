# Roadmap

## Phase 1 — Foundation (current)
- [x] Ollama integration with low-context wrapper
- [x] 3 simple agents (QA, report, prototype)
- [x] AgentGuard governance policy (monitor mode)
- [x] Script-based execution with cron support
- [x] Memory optimization placeholder

## Phase 2 — Hardening
- [ ] Switch agentguard.yaml to `enforce` mode
- [ ] Add AgentGuard CLI hooks to run-agent.sh (`agentguard pre/post`)
- [ ] Token budget tracking per agent per day
- [ ] Output quality scoring (simple heuristics)
- [ ] Error recovery and retry logic

## Phase 3 — Framework Integration
- [ ] **DeepAgents** — multi-step planning for complex tasks
  - Wire `adapters/deepagents.ts` to real SDK
  - Route via `agent-config.ts` framework field
  - Agent decomposition: break goals into sub-tasks
- [ ] **OpenCode** — interactive coding agent
  - Wire `adapters/opencode.ts` to real SDK
  - Tool-use governance via AgentGuard hooks
  - Sandbox file writes to `outputs/` only

## Phase 4 — Memory & Context
- [ ] **Google memory library** integration
  - Swap `config/memory.ts` stubs with real implementation
  - Rolling context window with summarization
  - Cross-session memory persistence
- [ ] Prompt caching for repeated patterns
- [ ] RAG over local codebase (lightweight)

## Phase 5 — Security
- [ ] **NVIDIA OpenShell** sandbox integration
  - Landlock + Seccomp isolation per agent run
  - Compile agentguard.yaml → OpenShell policy
  - See: AgentGuardHQ/agentguard#1036
- [ ] **Cisco DefenseClaw** scanning
  - Scan agent skills/plugins pre-install
  - MCP server verification

## Phase 6 — Scale
- [ ] Multi-model routing (qwen for fast, mistral for quality)
- [ ] Agent-to-agent communication (simple file-based)
- [ ] Cloud telemetry integration (AgentGuard Cloud)
- [ ] Dashboard for local swarm observability
