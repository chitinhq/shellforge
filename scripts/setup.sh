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

# ── 4. Chitin Kernel ──
header "4/8  🛡️  Chitin Kernel (Policy Enforcement)"
if command -v chitin &>/dev/null; then
  ok "Chitin kernel $(chitin --version 2>/dev/null || echo 'installed')"
elif [[ -f chitin.yaml ]]; then
  ok "Using built-in YAML evaluator (chitin.yaml found)"
  info "For full kernel (blast radius, personas): go install github.com/chitinhq/chitin/go/cmd/chitin@latest"
else
  warn "No chitin.yaml found — governance disabled"
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
  if [[ "$(uname)" == "Darwin" ]]; then
    info "OpenShell requires a Linux kernel (Landlock + Seccomp)."
    info "On macOS, it runs inside a Linux VM via Docker or Colima."
    echo ""

    # Check for Docker or Colima
    LINUX_VM=""
    if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
      ok "Docker detected — Linux VM available"
      LINUX_VM="docker"
    elif command -v colima &>/dev/null; then
      ok "Colima detected — Linux VM available"
      LINUX_VM="colima"
    elif command -v lima &>/dev/null; then
      ok "Lima detected — Linux VM available"
      LINUX_VM="lima"
    fi

    if [[ -z "$LINUX_VM" ]]; then
      ans=$(ask "Install a Linux VM for OpenShell? (Docker or Colima)" "y")
      if [[ "$ans" == "y" ]]; then
        echo ""
        echo "  Choose a Linux VM runtime:"
        echo "    1) Docker Desktop (recommended — most compatible)"
        echo "    2) Colima (lightweight, CLI-only, uses Lima)"
        read -rp "  Choose [1/2]: " vm_choice
        case "${vm_choice:-1}" in
          2)
            info "Installing Colima..."
            brew install colima docker
            colima start --cpu 2 --memory 4
            ok "Colima running (Linux VM with kernel sandbox support)"
            LINUX_VM="colima"
            ;;
          *)
            info "Install Docker Desktop from: https://www.docker.com/products/docker-desktop/"
            info "After install, restart this script."
            warn "Docker Desktop needs manual download — see link above"
            ;;
        esac
      fi
    fi

    if [[ -n "$LINUX_VM" ]]; then
      ans=$(ask "Install OpenShell inside Linux VM?" "y")
      if [[ "$ans" == "y" ]]; then
        info "Pulling OpenShell container..."
        docker pull nvidia/openshell:latest 2>/dev/null || {
          info "Building OpenShell from source..."
          docker run --rm -v "$(pwd)":/workspace -w /workspace ubuntu:22.04 bash -c             "apt-get update -qq && apt-get install -y -qq git make gcc >/dev/null 2>&1 &&              git clone --depth 1 https://github.com/NVIDIA/OpenShell /tmp/openshell 2>/dev/null &&              cd /tmp/openshell && make 2>/dev/null && cp openshell /workspace/.openshell-linux" 2>/dev/null
        }
        if [[ -f .openshell-linux ]]; then
          ok "OpenShell built (Linux binary at .openshell-linux)"
          info "Run sandboxed agents: docker run --rm -v \$(pwd):/workspace openshell ..."
        else
          ok "OpenShell container available via Docker"
        fi
      fi
    else
      warn "Skipped — no Linux VM. Install Docker or Colima to enable."
    fi

  else
    # Native Linux
    ans=$(ask "Install OpenShell? (kernel-level agent sandboxing)" "n")
    if [[ "$ans" == "y" ]]; then
      if [[ "$(uname -r | cut -d. -f1-2)" < "5.13" ]]; then
        warn "Kernel $(uname -r) — Landlock needs >= 5.13"
      fi
      info "Installing OpenShell..."
      git clone --depth 1 https://github.com/NVIDIA/OpenShell /tmp/openshell 2>/dev/null
      cd /tmp/openshell && make && sudo make install
      cd - >/dev/null
      if command -v openshell &>/dev/null; then
        ok "OpenShell installed"
      else
        warn "OpenShell build failed — check errors above"
      fi
    else
      warn "Skipped — agents run without kernel isolation"
    fi
  fi
fi

echo ""
echo -e "  ${BOLD}DefenseClaw${NC} — Cisco supply chain scanner"
if command -v defenseclaw &>/dev/null; then
  ok "DefenseClaw installed"
else
  ans=$(ask "Install DefenseClaw? (scan agent skills and MCP servers)" "n")
  if [[ "$ans" == "y" ]]; then
    if command -v pip3 &>/dev/null; then
      info "Attempting pip install..."
      pip3 install defenseclaw 2>/dev/null && ok "DefenseClaw installed" || {
        info "Not on pip — building from source..."
        git clone --depth 1 https://github.com/cisco/defenseclaw /tmp/defenseclaw 2>/dev/null
        if [[ -f /tmp/defenseclaw/Makefile ]]; then
          cd /tmp/defenseclaw && make && sudo make install
          cd - >/dev/null
          ok "DefenseClaw installed"
        else
          info "Install from: https://github.com/cisco/defenseclaw"
          warn "Manual install required — see link above"
        fi
      }
    else
      info "Install from: https://github.com/cisco/defenseclaw"
      warn "Manual install required — see link above"
    fi
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
