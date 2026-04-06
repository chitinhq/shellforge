# ShellForge — Copilot Instructions

> Copilot acts as **Tier C — Execution Workforce** in this repository.
> Implement well-specified issues, open draft PRs, never merge or approve.

## Project Overview

**ShellForge** is the execution harness for the Chitin platform — agent loop, LLM providers, tool-use, drift detection.

## Tech Stack

- **Language**: Go 1.22+
- **Module**: `github.com/chitinhq/shellforge`

## Build & Test

```bash
go build ./...
go test ./...
golangci-lint run
```

## Governance Rules

### DENY
- `git push` to main — always use feature branches
- `git force-push` — never rewrite shared history
- Write to `.env`, SSH keys, credentials

### ALWAYS
- Create feature branches: `agent/<type>/issue-<N>`
- Run `go build ./... && go test ./...` before creating PRs
- Link PRs to issues (`Closes #N`)

## PR Rules

- **NEVER merge PRs** — only Tier B or humans merge
- Max 300 lines changed per PR (soft limit)
- Always open as **draft PR** first
- If ambiguous, label `needs-spec` and stop

## Autonomy Directive

- **NEVER pause to ask for clarification** — make your best judgment
- If the issue is ambiguous, label it `needs-spec` and stop
