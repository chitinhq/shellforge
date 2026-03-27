# ShellForge — Terminal Bench 2.0 Adapter

Run ShellForge against [Terminal Bench 2.0](https://tbench.ai) via the [Harbor](https://github.com/harbor-framework/harbor) evaluation framework.

## Setup

```bash
# Install Harbor
pip install harbor

# Or with uv
uv tool install harbor
```

## Run

```bash
# Run against Terminal Bench 2.0 (local Docker)
harbor run -d terminal-bench@2.0 \
    --agent-import-path bench.agent:ShellForgeAgent \
    --env local -n 1

# With a specific model
SHELLFORGE_MODEL=qwen3:30b harbor run -d terminal-bench@2.0 \
    --agent-import-path bench.agent:ShellForgeAgent \
    --env local -n 1
```

## How It Works

1. Harbor spins up a Docker container per task
2. ShellForge is built from source inside the container
3. Ollama is installed and the model is pulled
4. `shellforge agent "<instruction>"` runs with full AgentGuard governance
5. Harbor's test harness checks results (pass/fail)

## Submit to Leaderboard

Email results to:
- mikeam@cs.stanford.edu
- alex@laude.org
