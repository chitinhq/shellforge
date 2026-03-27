/**
 * DeepAgents adapter — placeholder for future integration.
 *
 * DeepAgents provides multi-step planning and autonomous task decomposition.
 * When ready, this adapter translates between our simple agent interface
 * and DeepAgents' execution engine.
 *
 * Integration points:
 *   1. run-agent.sh can route to DeepAgents instead of direct Ollama calls
 *   2. Agent configs in agent-config.ts gain a `framework: 'deepagents'` option
 *   3. Memory layer (config/memory.ts) feeds into DeepAgents' context manager
 *
 * Install: npm install deepagents (when available)
 */

export interface DeepAgentTask {
  goal: string;
  constraints: string[];
  maxSteps: number;
  model: string;
}

export interface DeepAgentResult {
  success: boolean;
  artifacts: string[];
  steps: number;
  reasoning: string;
}

export async function runDeepAgent(_task: DeepAgentTask): Promise<DeepAgentResult> {
  throw new Error(
    'DeepAgents not installed. Install with: npm install deepagents\n' +
    'Then implement this adapter. See docs/roadmap.md for integration plan.'
  );
}
