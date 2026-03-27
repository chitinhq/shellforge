#!/usr/bin/env bash
# run-report-agent.sh — Run report agent, safe for cron
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec "$SCRIPT_DIR/run-agent.sh" report "$@"
