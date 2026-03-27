/**
 * OpenCode adapter — placeholder for future integration.
 *
 * OpenCode provides an interactive coding agent with tool use.
 * When ready, this adapter translates between our simple agent interface
 * and OpenCode's execution engine.
 *
 * Integration points:
 *   1. run-agent.sh can route to OpenCode for coding tasks
 *   2. AgentGuard governance hooks wrap OpenCode's tool calls
 *   3. Ollama serves as the local model backend for OpenCode
 *
 * Install: npm install opencode (when available)
 */

export interface OpenCodeTask {
  prompt: string;
  workingDir: string;
  allowedTools: string[];
  model: string;
}

export interface OpenCodeResult {
  success: boolean;
  filesChanged: string[];
  output: string;
}

export async function runOpenCode(_task: OpenCodeTask): Promise<OpenCodeResult> {
  throw new Error(
    'OpenCode not installed. Install with: npm install opencode\n' +
    'Then implement this adapter. See docs/roadmap.md for integration plan.'
  );
}
