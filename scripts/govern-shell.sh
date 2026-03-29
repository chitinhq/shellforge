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
    # Use jq --arg for safe JSON construction: handles quotes, backslashes, control chars in $COMMAND
    if command -v jq &>/dev/null; then
        JSON_PAYLOAD=$(jq -n --arg cmd "$COMMAND" '{"tool":"run_shell","action":$cmd,"path":"."}')
    else
        # Fallback: basic escaping (covers common cases; jq preferred)
        ESCAPED=$(printf '%s' "$COMMAND" | sed 's/\\/\\\\/g; s/"/\\"/g')
        JSON_PAYLOAD="{\"tool\":\"run_shell\",\"action\":\"$ESCAPED\",\"path\":\".\"}"
    fi
    RESULT=$(printf '%s' "$JSON_PAYLOAD" | shellforge evaluate 2>/dev/null || echo '{"allowed":false,"reason":"governance unavailable"}')

    if printf '%s' "$RESULT" | grep -q '"allowed":false'; then
        if command -v jq &>/dev/null; then
            REASON=$(printf '%s' "$RESULT" | jq -r '.reason // "policy violation"')
        else
            REASON=$(printf '%s' "$RESULT" | sed 's/.*"reason":"\([^"]*\)".*/\1/')
        fi
        echo "[AgentGuard] DENIED: $REASON" >&2
        exit 1
    fi

    exec "$REAL_SHELL" -c "$COMMAND"
else
    exec "$REAL_SHELL" "$@"
fi
