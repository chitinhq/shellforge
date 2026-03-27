#!/usr/bin/env bash
set -euo pipefail

# ─────────────────────────────────────────────────
# ShellForge Interactive Setup CLI
# Installs the full 8-project ecosystem step by step.
# Usage: bash scripts/setup.sh [--all | --minimal]
# ─────────────────────────────────────────────────

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

ok()   { echo -e "  ${GREEN}✓${NC} $1"; }
warn() { echo -e "  ${YELLOW}⚠${NC} $1"; }
fail() { echo -e "  ${RED}✗${NC} $1"; }
info() { echo -e "  ${BLUE}→${NC} $1"; }
header() { echo -e "\n${BOLD}$1${NC}"; }

ask() {
  local prompt="$1" default="${2:-y}"
  if [[ "$AUTO_ALL" == "1" ]]; then echo "y"; return; fi
  if [[ "$AUTO_MINIMAL" == "1" ]]; then echo "$default"; return; fi
  read -rp "  $prompt [y/n] ($default): " answer
  echo "${answer:-$default}"
}

AUTO_ALL=0
AUTO_MINIMAL=0
[[ "${1:-}" == "--all" ]] && AUTO_ALL=1
[[ "${1:-}" == "--minimal" ]] && AUTO_MINIMAL=1

echo ""
echo -e "${BOLD}🔥 ShellForge Setup${NC} — Interactive Ecosystem Installer"
echo "─────────────────────────────────────────────────"
echo ""

# ── 1. Core: Go ──
header "1/8  🔧 Go Toolchain"
if command -v go &>/dev/null; then
  ok "Go $(go version | awk '{print $3}')"
else
  fail "Go not found"
  ans=$(ask "Install Go?" "y")
  if [[ "$ans" == "y" ]]; then
    if [[ "$(uname)" == "Darwin" ]]; then
      brew install go
    else
      info "Download from https://go.dev/dl/"
      exit 1
    fi
  else
    fail "Go is required. Exiting."
    exit 1
  fi
fi

# Build the binary
if [[ -f go.mod ]]; then
  info "Building shellforge binary..."
  go build -o shellforge ./cmd/shellforge 2>/dev/null
  ok "Binary built: ./shellforge ($(du -h shellforge | awk '{print $1}'))"
fi

# ── 2. Ollama ──
header "2/8  🦙 Ollama (Local LLM Serving)"
if command -v ollama &>/dev/null; then
  ok "Ollama installed"
else
  ans=$(ask "Install Ollama? (required for local inference)" "y")
  if [[ "$ans" == "y" ]]; then
    if [[ "$(uname)" == "Darwin" ]]; then
      brew install ollama
    else
      curl -fsSL https://ollama.ai/install.sh | sh
    fi
    ok "Ollama installed"
  else
    warn "Skipped — agents won't work without Ollama"
  fi
fi

if command -v ollama &>/dev/null; then
  if curl -sf http://localhost:11434/api/tags &>/dev/null; then
    ok "Ollama running"
  else
    info "Starting Ollama..."
    ollama serve &>/dev/null &
    sleep 3
    if curl -sf http://localhost:11434/api/tags &>/dev/null; then
      ok "Ollama running"
    else
      warn "Ollama not responding — start manually: ollama serve"
    fi
  fi

  MODEL="${OLLAMA_MODEL:-qwen3:1.7b}"
  if ollama list 2>/dev/null | grep -q "$MODEL"; then
    ok "Model ready: $MODEL"
  else
    ans=$(ask "Pull model $MODEL? (~1GB)" "y")
    if [[ "$ans" == "y" ]]; then
      ollama pull "$MODEL"
      ok "Model ready: $MODEL"
    fi
  fi
fi

# ── 3. RTK ──
header "3/8  ⚡ RTK — Rust Token Killer (60-90% token savings)"
if command -v rtk &>/dev/null; then
  ok "RTK $(rtk --version 2>/dev/null || echo 'installed')"
else
  ans=$(ask "Install RTK? (compresses shell output before LLM sees it)" "y")
  if [[ "$ans" == "y" ]]; then
    if [[ "$(uname)" == "Darwin" ]]; then
      brew install rtk-ai/tap/rtk
    else
      curl -fsSL https://raw.githubusercontent.com/rtk-ai/rtk/main/install.sh | sh
    fi
    if command -v rtk &>/dev/null; then
      ok "RTK installed"
    else
      warn "RTK install may need PATH update. Check: https://github.com/rtk-ai/rtk"
    fi
  else
    warn "Skipped — shell commands will use full token output"
  fi
fi

# ── 4. AgentGuard Kernel ──
header "4/8  🛡️  AgentGuard Kernel (Policy Enforcement)"
if command -v agentguard &>/dev/null; then
  ok "AgentGuard kernel $(agentguard --version 2>/dev/null || echo 'installed')"
elif [[ -f agentguard.yaml ]]; then
  ok "Using built-in YAML evaluator (agentguard.yaml found)"
  info "For full kernel (blast radius, personas): go install github.com/AgentGuardHQ/agent-guard/go/cmd/agentguard@latest"
else
  warn "No agentguard.yaml found — governance disabled"
fi

# ── 5. OpenCode ──
header "5/8  🤖 OpenCode (AI Coding Agent Engine)"
if command -v opencode &>/dev/null; then
  ok "OpenCode installed"
else
  ans=$(ask "Install OpenCode? (Go-native coding agent with tool use)" "n")
  if [[ "$ans" == "y" ]]; then
    if [[ "$(uname)" == "Darwin" ]]; then
      brew install opencode
    elif command -v npm &>/dev/null; then
      npm install -g opencode-ai
    else
      curl -fsSL https://opencode.ai/install | bash
    fi
    if command -v opencode &>/dev/null; then
      ok "OpenCode installed"
    else
      warn "OpenCode install needs PATH update"
    fi
  else
    warn "Skipped — using native Ollama engine (built-in)"
  fi
fi

# ── 6. DeepAgents ──
header "6/8  🧠 DeepAgents (Multi-Step Planning via LangGraph)"
DA_INSTALLED=0
if command -v node &>/dev/null && node -e "require('deepagents')" 2>/dev/null; then
  ok "DeepAgents (npm)"
  DA_INSTALLED=1
elif python3 -c "import deepagents" 2>/dev/null; then
  ok "DeepAgents (Python)"
  DA_INSTALLED=1
fi

if [[ "$DA_INSTALLED" == "0" ]]; then
  ans=$(ask "Install DeepAgents? (multi-step task planning)" "n")
  if [[ "$ans" == "y" ]]; then
    echo ""
    echo "  Install via:"
    echo "    1) npm install deepagents"
    echo "    2) pip install deepagents"
    read -rp "  Choose [1/2]: " choice
    if [[ "${choice:-1}" == "2" ]]; then
      pip3 install deepagents
    else
      npm install deepagents
    fi
    ok "DeepAgents installed"
  else
    warn "Skipped — using native Ollama engine"
  fi
fi

# ── 7. TurboQuant ──
header "7/8  🧠 TurboQuant (Google KV Cache Compression — 6x memory savings)"
if python3 -c "import turboquant_pytorch" 2>/dev/null || python3 -c "import turboquant" 2>/dev/null; then
  ok "TurboQuant installed"
else
  ans=$(ask "Install TurboQuant? (run 14B models on 8GB Macs)" "n")
  if [[ "$ans" == "y" ]]; then
    pip3 install turboquant-pytorch
    if python3 -c "import turboquant_pytorch" 2>/dev/null; then
      ok "TurboQuant installed"
    else
      warn "TurboQuant install failed — check pip output"
    fi
  else
    warn "Skipped — using standard Ollama quantization"
  fi
fi

# ── 8. Security: OpenShell + DefenseClaw ──
header "8/8  🔒 Security Layer"

echo -e "  ${BOLD}OpenShell${NC} — NVIDIA kernel sandbox (Landlock + Seccomp)"
if command -v openshell &>/dev/null; then
  ok "OpenShell installed"
else
  ans=$(ask "Install OpenShell? (kernel-level agent sandboxing)" "n")
  if [[ "$ans" == "y" ]]; then
    info "Clone and build from https://github.com/NVIDIA/OpenShell"
    info "  git clone https://github.com/NVIDIA/OpenShell && cd OpenShell && make install"
    warn "Manual install required — see link above"
  else
    warn "Skipped — agents run without kernel isolation"
  fi
fi

echo ""
echo -e "  ${BOLD}DefenseClaw${NC} — Cisco supply chain scanner"
if command -v defenseclaw &>/dev/null; then
  ok "DefenseClaw installed"
else
  ans=$(ask "Install DefenseClaw? (scan skills and MCP servers)" "n")
  if [[ "$ans" == "y" ]]; then
    info "Install from https://github.com/cisco/defenseclaw"
    warn "Manual install required — see link above"
  else
    warn "Skipped — no supply chain scanning"
  fi
fi

# ── Output dirs ──
mkdir -p outputs/logs outputs/reports

# ── Summary ──
echo ""
echo "─────────────────────────────────────────────────"
echo -e "${BOLD}🔥 ShellForge Setup Complete${NC}"
echo "─────────────────────────────────────────────────"
echo ""

if [[ -f ./shellforge ]]; then
  ./shellforge status
fi

echo ""
echo -e "${BOLD}Quick start:${NC}"
echo "  ./shellforge qa              # analyze code for test gaps"
echo "  ./shellforge report          # weekly status report"
echo '  ./shellforge agent "prompt"  # run any task with tool use'
echo "  ./shellforge status          # ecosystem health check"
echo "  ./shellforge scan            # supply chain scan"
echo ""
echo -e "  ${BLUE}--all${NC}       Install everything non-interactively"
echo -e "  ${BLUE}--minimal${NC}   Install only core (Ollama + Go)"
echo ""
