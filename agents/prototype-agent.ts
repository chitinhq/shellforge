/**
 * Prototype Agent — generates code snippets or scaffolds from a prompt.
 *
 * Usage: tsx agents/prototype-agent.ts "create a REST API health endpoint"
 * Output: outputs/logs/prototype-<timestamp>.log
 */
import { writeFileSync, mkdirSync } from 'fs';
import { resolve } from 'path';
import { generate, isOllamaRunning } from '../config/ollama.js';
import { getAgent } from '../config/agent-config.js';
import { initMemoryLayer } from '../config/memory.js';

const ROOT = resolve(import.meta.dirname, '..');

async function main() {
  const config = getAgent('prototype');
  const userPrompt = process.argv[2];

  if (!userPrompt) {
    console.error('Usage: tsx agents/prototype-agent.ts "<prompt>"');
    console.error('Example: tsx agents/prototype-agent.ts "create a REST API health endpoint in Express"');
    process.exit(1);
  }

  console.log(`[prototype-agent] starting — prompt: "${userPrompt.slice(0, 80)}..."`);

  if (!(await isOllamaRunning())) {
    console.error('[prototype-agent] ERROR: Ollama is not running. Start it with: ollama serve');
    process.exit(1);
  }

  await initMemoryLayer();

  console.log('[prototype-agent] generating...');
  const result = await generate({
    prompt: userPrompt,
    model: config.model,
    system: config.system,
    temperature: 0.2,
  });

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  mkdirSync(config.outputDir, { recursive: true });
  const outPath = resolve(config.outputDir, `prototype-${timestamp}.log`);

  const output = [
    `# Prototype — ${new Date().toISOString()}`,
    `**Prompt:** ${userPrompt}`,
    `**Model:** ${result.model}`,
    `**Tokens:** ${result.promptTokens} prompt + ${result.responseTokens} response`,
    '',
    result.text,
  ].join('\n');

  writeFileSync(outPath, output);
  console.log(`[prototype-agent] done — output: ${outPath}`);
}

main().catch(err => {
  console.error('[prototype-agent] FATAL:', err.message);
  process.exit(1);
});
