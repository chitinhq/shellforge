#!/usr/bin/env bash
# run-qa-agent.sh — Run QA agent, safe for cron
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec "$SCRIPT_DIR/run-agent.sh" qa "$@"
