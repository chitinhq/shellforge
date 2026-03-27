/**
 * QA Agent — analyzes source files and suggests tests/findings.
 *
 * Usage: tsx agents/qa-agent.ts [file-or-directory]
 * Output: outputs/logs/qa-<timestamp>.log
 */
import { readFileSync, writeFileSync, readdirSync, statSync, mkdirSync } from 'fs';
import { resolve, extname } from 'path';
import { generate, isOllamaRunning } from '../config/ollama.js';
import { getAgent } from '../config/agent-config.js';
import { initMemoryLayer } from '../config/memory.js';

const ROOT = resolve(import.meta.dirname, '..');

async function main() {
  const config = getAgent('qa');
  const target = process.argv[2] || resolve(ROOT, 'agents');

  console.log(`[qa-agent] starting — target: ${target}`);

  if (!(await isOllamaRunning())) {
    console.error('[qa-agent] ERROR: Ollama is not running. Start it with: ollama serve');
    process.exit(1);
  }

  await initMemoryLayer();

  const files = collectFiles(target, ['.ts', '.js', '.py', '.go']);
  if (files.length === 0) {
    console.log('[qa-agent] no source files found');
    process.exit(0);
  }

  // Limit to first 5 files to stay within context budget
  const batch = files.slice(0, 5);
  const code = batch.map(f => `--- ${f} ---\n${readFileSync(f, 'utf8')}`).join('\n\n');

  const prompt = `Analyze these source files and suggest specific test cases for each.\n\n${code}`;

  console.log(`[qa-agent] analyzing ${batch.length} files...`);
  const result = await generate({
    prompt,
    model: config.model,
    system: config.system,
  });

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  mkdirSync(config.outputDir, { recursive: true });
  const outPath = resolve(config.outputDir, `qa-${timestamp}.log`);

  const output = [
    `# QA Agent Report — ${new Date().toISOString()}`,
    `**Files analyzed:** ${batch.map(f => f.replace(ROOT + '/', '')).join(', ')}`,
    `**Model:** ${result.model}`,
    `**Tokens:** ${result.promptTokens} prompt + ${result.responseTokens} response`,
    '',
    result.text,
  ].join('\n');

  writeFileSync(outPath, output);
  console.log(`[qa-agent] done — output: ${outPath}`);
}

function collectFiles(target: string, extensions: string[]): string[] {
  const stat = statSync(target);
  if (stat.isFile()) return extensions.includes(extname(target)) ? [target] : [];

  const results: string[] = [];
  for (const entry of readdirSync(target, { withFileTypes: true })) {
    if (entry.name.startsWith('.') || entry.name === 'node_modules') continue;
    const full = resolve(target, entry.name);
    if (entry.isFile() && extensions.includes(extname(entry.name))) {
      results.push(full);
    } else if (entry.isDirectory()) {
      results.push(...collectFiles(full, extensions));
    }
  }
  return results;
}

main().catch(err => {
  console.error('[qa-agent] FATAL:', err.message);
  process.exit(1);
});
