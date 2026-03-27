#!/usr/bin/env bash
set -euo pipefail

echo "=== ShellForge Setup ==="

# Check Go
if command -v go &>/dev/null; then
  echo "✓ Go $(go version | awk '{print $3}')"
else
  echo "✗ Go not found. Install: https://go.dev/dl/"
  exit 1
fi

# Build the binary
echo "→ Building shellforge..."
go build -o shellforge ./cmd/shellforge
echo "✓ Binary built: ./shellforge"

# Check/install Ollama
if command -v ollama &>/dev/null; then
  echo "✓ Ollama installed"
else
  echo "→ Installing Ollama..."
  if [[ "$(uname)" == "Darwin" ]]; then
    brew install ollama
  else
    curl -fsSL https://ollama.ai/install.sh | sh
  fi
fi

# Start Ollama if not running
if ! curl -sf http://localhost:11434/api/tags &>/dev/null; then
  echo "→ Starting Ollama..."
  ollama serve &>/dev/null &
  sleep 3
fi

if curl -sf http://localhost:11434/api/tags &>/dev/null; then
  echo "✓ Ollama running"
else
  echo "⚠ Ollama not responding — start manually: ollama serve"
fi

# Pull model
MODEL="${OLLAMA_MODEL:-qwen3:1.7b}"
echo "→ Pulling model ${MODEL}..."
ollama pull "$MODEL"
echo "✓ Model ready: ${MODEL}"

# Create output dirs
mkdir -p outputs/logs outputs/reports
echo "✓ Output directories ready"

# Show ecosystem status
echo ""
./shellforge status

echo ""
echo "=== ShellForge Setup Complete ==="
echo ""
echo "Quick start:"
echo "  ./shellforge qa          # analyze code"
echo "  ./shellforge report      # weekly report"
echo "  ./shellforge agent \"build a health endpoint\""
