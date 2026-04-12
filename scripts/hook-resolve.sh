#!/bin/bash
# hook-resolve.sh — Universal Chitin binary resolver for all drivers.
# Ensures governance hooks + telemetry work in worktrees, local installs, and global installs.
#
# Usage (from any hook config):
#   source scripts/hook-resolve.sh
#   eval "$CHITIN_BIN claude-hook"   # or copilot-hook, codex-hook, gemini-hook
#
# Sets:
#   CHITIN_BIN — shell command prefix that works everywhere (may include cd)
#   _CH_MAIN_ROOT  — path to the main (non-worktree) checkout

# Resolve project root
_CH_PROJECT_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"

# Source persona env if available
if [ -f "$_CH_PROJECT_ROOT/.chitin/persona.env" ]; then
  set -a; source "$_CH_PROJECT_ROOT/.chitin/persona.env"; set +a
fi

# Source workspace .env for telemetry config (API key, cloud endpoint, tenant ID)
_CH_WS_ROOT="$HOME/chitin-workspace"
if [ -f "$_CH_WS_ROOT/.env" ]; then
  set -a; source "$_CH_WS_ROOT/.env"; set +a
fi

# Find the main worktree root (where node_modules lives)
_CH_MAIN_ROOT="$(git worktree list --porcelain 2>/dev/null | sed -n '1s/^worktree //p')"
_CH_IN_WORKTREE=0
if [ -n "$_CH_MAIN_ROOT" ] && [ "$_CH_MAIN_ROOT" != "$_CH_PROJECT_ROOT" ]; then
  _CH_IN_WORKTREE=1
fi

# Resolve binary — priority: local dev > global install > main worktree fallback
CHITIN_BIN=""

# 1. Global install (npm install -g @red-codes/chitin)
#    Works in any directory — no worktree issues.
if command -v chitin &>/dev/null; then
  CHITIN_BIN="chitin"
fi

# 2. Local dev (apps/cli/dist/bin.js in current or main worktree)
#    ESM resolution requires CWD to be where node_modules exists.
#    In worktrees, we MUST cd to the main root before running the binary.
if [ -f "$_CH_PROJECT_ROOT/apps/cli/dist/bin.js" ]; then
  if [ "$_CH_IN_WORKTREE" -eq 1 ] && [ -n "$_CH_MAIN_ROOT" ]; then
    # Worktree: run from main root for ESM package resolution
    CHITIN_BIN="cd $_CH_MAIN_ROOT && node apps/cli/dist/bin.js"
  else
    CHITIN_BIN="node $_CH_PROJECT_ROOT/apps/cli/dist/bin.js"
  fi
elif [ "$_CH_IN_WORKTREE" -eq 1 ] && [ -n "$_CH_MAIN_ROOT" ] && [ -f "$_CH_MAIN_ROOT/apps/cli/dist/bin.js" ]; then
  # Worktree doesn't have the binary but main root does
  CHITIN_BIN="cd $_CH_MAIN_ROOT && node apps/cli/dist/bin.js"
fi

# 3. Probe: verify the resolved binary actually works
if [ -n "$CHITIN_BIN" ]; then
  if ! eval "$CHITIN_BIN --version" >/dev/null 2>&1; then
    # Binary fails — try main worktree as last resort
    if [ -n "$_CH_MAIN_ROOT" ] && [ -f "$_CH_MAIN_ROOT/apps/cli/dist/bin.js" ]; then
      CHITIN_BIN="cd $_CH_MAIN_ROOT && node apps/cli/dist/bin.js"
      if ! eval "$CHITIN_BIN --version" >/dev/null 2>&1; then
        CHITIN_BIN=""  # give up, bootstrap exemption will handle it
      fi
    else
      CHITIN_BIN=""
    fi
  fi
fi

export CHITIN_BIN
