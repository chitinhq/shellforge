#!/bin/bash
# govern-shell.sh — Governed shell for ShellForge
#
# Every command is evaluated by AgentGuard before running.
# Set as SHELL for governed agents (Goose, etc.)

set -euo pipefail

REAL_SHELL="${SHELLFORGE_REAL_SHELL:-/bin/bash}"

# Fall through if shellforge not installed or no policy
if ! command -v shellforge &>/dev/null; then
    exec "$REAL_SHELL" "$@"
fi
if [ ! -f "agentguard.yaml" ] && [ ! -f "../agentguard.yaml" ]; then
    exec "$REAL_SHELL" "$@"
fi

# Handle -c flag (how subprocesses call shells)
if [ "${1:-}" = "-c" ]; then
    shift
    COMMAND="$*"

    # Evaluate through AgentGuard
    RESULT=$(printf '{"tool":"run_shell","action":"%s","path":"."}' "$COMMAND" | shellforge evaluate 2>/dev/null || echo '{"allowed":true}')
    
    if echo "$RESULT" | grep -q '"allowed":false'; then
        REASON=$(echo "$RESULT" | sed 's/.*"reason":"\([^"]*\)".*/\1/')
        echo "[AgentGuard] DENIED: $REASON" >&2
        exit 1
    fi

    exec "$REAL_SHELL" -c "$COMMAND"
else
    exec "$REAL_SHELL" "$@"
fi
