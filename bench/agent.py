"""ShellForge agent adapter for Harbor / Terminal Bench 2.0.

Usage:
    harbor run -d terminal-bench@2.0 \
        --agent-import-path bench.agent:ShellForgeAgent \
        --env local -n 1
"""

import os

from harbor.agent import BaseInstalledAgent, AgentContext
from harbor.environment import BaseEnvironment


class ShellForgeAgent(BaseInstalledAgent):
    """ShellForge — governed agent runtime for Terminal Bench.

    Installs the ShellForge Go binary + Ollama inside the container,
    then runs `shellforge agent` with AgentGuard governance on every task.
    """

    SHELLFORGE_VERSION = "latest"
    OLLAMA_MODEL = os.environ.get("SHELLFORGE_MODEL", "qwen3:30b")

    def name(self) -> str:
        return "shellforge"

    def version(self) -> str:
        return "0.2.0"

    def install(self) -> None:
        """Install ShellForge + Ollama inside the container."""
        # Install Go (if not present)
        self.exec_as_root(
            "command -v go || ("
            "  curl -fsSL https://go.dev/dl/go1.23.0.linux-amd64.tar.gz | tar -C /usr/local -xz &&"
            "  ln -sf /usr/local/go/bin/go /usr/local/bin/go"
            ")"
        )

        # Install Ollama
        self.exec_as_root(
            "command -v ollama || curl -fsSL https://ollama.ai/install.sh | sh"
        )

        # Clone and build ShellForge
        self.exec_as_agent(
            "cd /tmp &&"
            " git clone --depth 1 https://github.com/chitinhq/shellforge.git &&"
            " cd shellforge &&"
            " go build -o /usr/local/bin/shellforge ./cmd/shellforge/"
        )

        # Copy governance policy
        self.exec_as_agent(
            "cp /tmp/shellforge/agentguard.yaml /home/agent/agentguard.yaml"
        )

        # Start Ollama and pull model
        self.exec_as_root("ollama serve &")
        self.exec_as_agent("sleep 3 && ollama pull " + self.OLLAMA_MODEL)

    def run(self, instruction: str, environment: BaseEnvironment,
            context: AgentContext) -> None:
        """Run ShellForge agent on the task instruction."""
        # Escape the instruction for shell safety
        escaped = instruction.replace("'", "'\\''")

        # Run shellforge agent with the task instruction
        # Governance (agentguard.yaml) wraps every tool call
        result = environment.exec(
            "cd /home/agent && shellforge agent '" + escaped + "'",
            timeout=600,
        )

        # Store trajectory for leaderboard submission
        if result.stdout:
            context.trajectory = result.stdout
        if result.stderr:
            context.metadata["stderr"] = result.stderr[-2000:]

    def populate_context_post_run(self, context: AgentContext) -> None:
        """Extract final state for scoring."""
        pass  # test.sh handles verification
