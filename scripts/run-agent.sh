#!/usr/bin/env bash
# run-agent.sh — Generic agent runner with AgentGuard governance
# Usage: run-agent.sh <agent-name> [extra args...]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

AGENT_NAME="${1:?Usage: run-agent.sh <qa|report|prototype> [args...]}"
shift

# Load .env
if [ -f .env ]; then
  set -a; source .env; set +a
fi

TIMEOUT="${AGENT_TIMEOUT:-300}"
TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
LOG_FILE="outputs/logs/${AGENT_NAME}-run-${TIMESTAMP}.log"
mkdir -p outputs/logs outputs/reports

echo "[$(date)] START: $AGENT_NAME (timeout=${TIMEOUT}s)" | tee -a "$LOG_FILE"

# AgentGuard pre-hook (if installed)
if command -v agentguard &>/dev/null; then
  echo "[$(date)] AgentGuard governance: active" >> "$LOG_FILE"
else
  echo "[$(date)] AgentGuard governance: not installed (monitor only via agentguard.yaml)" >> "$LOG_FILE"
fi

# Route to agent
case "$AGENT_NAME" in
  qa)       AGENT_FILE="agents/qa-agent.ts" ;;
  report)   AGENT_FILE="agents/report-agent.ts" ;;
  prototype) AGENT_FILE="agents/prototype-agent.ts" ;;
  *)
    echo "Unknown agent: $AGENT_NAME (available: qa, report, prototype)" >&2
    exit 1
    ;;
esac

# Run with timeout
timeout "$TIMEOUT" npx tsx "$AGENT_FILE" "$@" 2>&1 | tee -a "$LOG_FILE"
EXIT_CODE=${PIPESTATUS[0]}

echo "[$(date)] FINISH: $AGENT_NAME exit_code=$EXIT_CODE" | tee -a "$LOG_FILE"
exit "$EXIT_CODE"
