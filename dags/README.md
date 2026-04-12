# ShellForge DAGs — Dagu Workflow Definitions

Dagu orchestrates ShellForge agent runs as DAG workflows with scheduling, dependencies, retries, and a web dashboard.

## Setup

### Mac
```bash
brew install dagu
```

### Linux
```bash
curl -sL https://raw.githubusercontent.com/dagu-org/dagu/main/scripts/installer.sh | bash
```

## Usage

```bash
# Start Dagu server (web UI at http://localhost:8080)
dagu server --dags=./dags

# Run a specific DAG now
dagu start dags/sdlc-swarm.yaml

# Dry run (show what would execute)
dagu dry dags/sdlc-swarm.yaml
```

## Available DAGs

| DAG | Schedule | What It Does |
|-----|----------|--------------|
| `sdlc-swarm.yaml` | Daily 9am | QA analysis → security scan → daily report |
| `studio-swarm.yaml` | Every 4h | Issues → tests → deps → health report |

## Custom DAGs

Create your own in this directory:

```yaml
# my-workflow.yaml
schedule: "0 */6 * * *"  # every 6 hours

steps:
  - name: my-agent
    command: shellforge agent "your task here"
    dir: /path/to/your/repo
```

Every `shellforge agent` call is governed by `chitin.yaml` in the target directory.

## Dagu Dashboard

Start with `dagu server --dags=./dags` and open http://localhost:8080 to see:
- DAG execution history
- Step-by-step logs
- Retry controls
- Schedule overview
