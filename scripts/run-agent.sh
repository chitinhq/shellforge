#!/usr/bin/env bash
set -euo pipefail
# Generic agent runner for cron. Usage: run-agent.sh <command> [args...]
# Example: run-agent.sh qa agents/
cd "$(dirname "$0")/.."

COMMAND="${1:-qa}"
shift || true

if [[ ! -f ./shellforge ]]; then
  echo "[run-agent] Building shellforge..."
  go build -o shellforge ./cmd/shellforge
fi

./shellforge "$COMMAND" "$@"
