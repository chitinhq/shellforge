import { existsSync, readFileSync } from 'fs';
import { resolve } from 'path';

/** Configuration for a single agent, resolved from env vars and defaults. */
export interface AgentConfig {
  name: string;
  description: string;
  timeout: number;
  outputDir: string;
  model: string;
  system?: string;
}

const ROOT = resolve(import.meta.dirname, '..');
const OUTPUT_DIR = process.env.AGENT_OUTPUT_DIR || 'outputs';
const DEFAULT_TIMEOUT = parseInt(process.env.AGENT_TIMEOUT || '300', 10);
const DEFAULT_MODEL = process.env.OLLAMA_MODEL || 'qwen3:1.7b';

export const agents: Record<string, AgentConfig> = {
  qa: {
    name: 'qa-agent',
    description: 'Analyzes code and suggests tests or findings',
    timeout: DEFAULT_TIMEOUT,
    outputDir: resolve(ROOT, OUTPUT_DIR, 'logs'),
    model: DEFAULT_MODEL,
    system: 'You are a QA engineer. Analyze the provided code and suggest specific, actionable test cases. Be concise.',
  },
  report: {
    name: 'report-agent',
    description: 'Generates weekly-style markdown summary from git log and agent activity',
    timeout: DEFAULT_TIMEOUT,
    outputDir: resolve(ROOT, OUTPUT_DIR, 'reports'),
    model: DEFAULT_MODEL,
    system: 'You are a technical writer. Generate a concise markdown report summarizing the provided activity data. Use headers, bullets, and keep it under 500 words.',
  },
  prototype: {
    name: 'prototype-agent',
    description: 'Generates small code snippets or scaffolds from a prompt',
    timeout: DEFAULT_TIMEOUT,
    outputDir: resolve(ROOT, OUTPUT_DIR, 'logs'),
    model: DEFAULT_MODEL,
    system: 'You are a senior engineer. Generate clean, minimal code for the requested task. Include brief comments. Output code only, no explanation.',
  },
};

/**
 * Retrieve the resolved configuration for a named agent.
 *
 * @param name - Agent key (e.g. `'qa'`, `'report'`, `'prototype'`).
 * @returns The agent's `AgentConfig`.
 * @throws If no agent with the given name is registered.
 */
export function getAgent(name: string): AgentConfig {
  const config = agents[name];
  if (!config) {
    const available = Object.keys(agents).join(', ');
    throw new Error(`Unknown agent "${name}". Available: ${available}`);
  }
  return config;
}

/**
 * Load environment variables from `.env` into `process.env`.
 * Existing env vars are never overwritten. Silently no-ops if `.env` is absent.
 */
export function loadEnv(): void {
  const envPath = resolve(ROOT, '.env');
  if (!existsSync(envPath)) return;

  const content = readFileSync(envPath, 'utf8');
  for (const line of content.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const eq = trimmed.indexOf('=');
    if (eq < 0) continue;
    const key = trimmed.slice(0, eq).trim();
    const val = trimmed.slice(eq + 1).trim();
    if (!process.env[key]) process.env[key] = val;
  }
}
