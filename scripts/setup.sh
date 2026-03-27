#!/usr/bin/env bash
# setup.sh — Install dependencies and configure local swarm
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

echo "=== ShellForge Setup ==="
echo ""

# 1. Node.js
if ! command -v node &>/dev/null; then
  echo "ERROR: Node.js not found. Install Node 20+ first."
  exit 1
fi
NODE_VER=$(node -v | sed 's/v//' | cut -d. -f1)
if [ "$NODE_VER" -lt 20 ]; then
  echo "ERROR: Node $NODE_VER found, need 20+."
  exit 1
fi
echo "✓ Node.js $(node -v)"

# 2. Install npm deps
if [ ! -d node_modules ]; then
  echo "→ Installing npm dependencies..."
  npm install --quiet
else
  echo "✓ npm dependencies installed"
fi

# 3. Ollama
if ! command -v ollama &>/dev/null; then
  echo ""
  echo "→ Installing Ollama..."
  if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "  Download from: https://ollama.com/download/mac"
    echo "  Or: brew install ollama"
    echo ""
    read -p "  Install via brew? [Y/n] " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Nn]$ ]]; then
      brew install ollama
    else
      echo "  Please install Ollama manually and re-run setup.sh"
      exit 1
    fi
  elif [[ "$OSTYPE" == "linux"* ]]; then
    curl -fsSL https://ollama.com/install.sh | sh
  else
    echo "  Unsupported OS. Install Ollama from: https://ollama.com"
    exit 1
  fi
fi
echo "✓ Ollama $(ollama --version 2>/dev/null || echo 'installed')"

# 4. Start Ollama if not running
if ! curl -sf http://localhost:11434/api/tags &>/dev/null; then
  echo "→ Starting Ollama..."
  ollama serve &>/dev/null &
  sleep 3
  if ! curl -sf http://localhost:11434/api/tags &>/dev/null; then
    echo "  WARNING: Ollama didn't start. Run 'ollama serve' manually."
  else
    echo "✓ Ollama running"
  fi
else
  echo "✓ Ollama running"
fi

# 5. Pull default model
MODEL="${OLLAMA_MODEL:-qwen3:1.7b}"
echo "→ Pulling model: $MODEL (this may take a few minutes on first run)..."
ollama pull "$MODEL" 2>/dev/null || echo "  WARNING: Could not pull $MODEL. Run: ollama pull $MODEL"
echo "✓ Model ready: $MODEL"

# 6. Create .env if missing
if [ ! -f .env ]; then
  cp .env.example .env
  echo "✓ Created .env from .env.example"
else
  echo "✓ .env exists"
fi

# 7. Create output dirs
mkdir -p outputs/reports outputs/logs
touch outputs/reports/.gitkeep outputs/logs/.gitkeep
echo "✓ Output directories ready"

echo ""
echo "=== ShellForge Setup Complete ==="
echo ""
echo "Next steps:"
echo "  1. Edit .env if you want to change the model or settings"
echo "  2. Run an agent:"
echo "     npm run report              # generate a status report"
echo "     npm run qa                  # analyze code for test suggestions"
echo "     npm run agent -- 'prompt'   # prototype code from a prompt"
echo "  3. Or use scripts:"
echo "     scripts/run-report-agent.sh"
echo "     scripts/run-qa-agent.sh"
echo "     scripts/run-agent.sh qa"
echo ""
