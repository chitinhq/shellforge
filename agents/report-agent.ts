/**
 * Report Agent — generates a markdown summary from git log and agent activity.
 *
 * Usage: tsx agents/report-agent.ts [repo-path]
 * Output: outputs/reports/report-<timestamp>.md
 */
import { writeFileSync, readdirSync, readFileSync, mkdirSync } from 'fs';
import { resolve } from 'path';
import { execSync } from 'child_process';
import { generate, isOllamaRunning } from '../config/ollama.js';
import { getAgent } from '../config/agent-config.js';
import { initMemoryLayer } from '../config/memory.js';

const ROOT = resolve(import.meta.dirname, '..');

async function main() {
  const config = getAgent('report');
  const repoPath = process.argv[2] || ROOT;

  console.log(`[report-agent] starting — repo: ${repoPath}`);

  if (!(await isOllamaRunning())) {
    console.error('[report-agent] ERROR: Ollama is not running. Start it with: ollama serve');
    process.exit(1);
  }

  await initMemoryLayer();

  // Gather git log (last 7 days)
  let gitLog = '(no git history available)';
  try {
    gitLog = execSync(
      'git log --oneline --since="7 days ago" --no-merges --max-count=50',
      { cwd: repoPath, encoding: 'utf8', timeout: 10000 }
    ).trim() || '(no commits in last 7 days)';
  } catch {
    console.log('[report-agent] warning: could not read git log');
  }

  // Gather recent agent logs
  const logsDir = resolve(ROOT, 'outputs/logs');
  let recentLogs = '(no agent logs found)';
  try {
    const logFiles = readdirSync(logsDir)
      .filter(f => f.endsWith('.log'))
      .sort()
      .slice(-5);
    if (logFiles.length > 0) {
      recentLogs = logFiles.map(f => {
        const content = readFileSync(resolve(logsDir, f), 'utf8');
        const preview = content.split('\n').slice(0, 10).join('\n');
        return `--- ${f} ---\n${preview}`;
      }).join('\n\n');
    }
  } catch { /* no logs yet */ }

  const prompt = `Generate a concise weekly status report in markdown based on this data.

## Git Activity (last 7 days)
${gitLog}

## Recent Agent Logs
${recentLogs}

Include sections: Summary, Key Changes, Agent Activity, Recommendations.`;

  console.log('[report-agent] generating report...');
  const result = await generate({
    prompt,
    model: config.model,
    system: config.system,
  });

  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  mkdirSync(config.outputDir, { recursive: true });
  const outPath = resolve(config.outputDir, `report-${timestamp}.md`);

  const output = [
    `# Weekly Status Report`,
    `> Generated: ${new Date().toISOString()}`,
    `> Model: ${result.model} | Tokens: ${result.promptTokens}+${result.responseTokens}`,
    '',
    result.text,
  ].join('\n');

  writeFileSync(outPath, output);
  console.log(`[report-agent] done — output: ${outPath}`);
}

main().catch(err => {
  console.error('[report-agent] FATAL:', err.message);
  process.exit(1);
});
