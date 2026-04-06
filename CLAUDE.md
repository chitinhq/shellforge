## Agent Identity

At session start, if you see `[Chitin] No agent identity set`, ask the user:
1. **Role**: developer / reviewer / ops / security / planner
2. **Driver**: human / claude-code / copilot / ci

Then run: `scripts/write-persona.sh <driver> <role>`

## Project

ShellForge is the execution harness for the Chitin platform — agent loop, LLM providers, tool-use, drift detection.

**Module**: `github.com/chitinhq/shellforge`
**Language**: Go 1.22+

## Build

```bash
go build ./...
go test ./...
golangci-lint run
```
