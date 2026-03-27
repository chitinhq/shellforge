import { optimizePrompt, trackUsage } from './memory.js';

const OLLAMA_HOST = process.env.OLLAMA_HOST || 'http://localhost:11434';
const OLLAMA_MODEL = process.env.OLLAMA_MODEL || 'qwen3:1.7b';
const OLLAMA_CTX_SIZE = parseInt(process.env.OLLAMA_CTX_SIZE || '4096', 10);

/** Options for a generation request sent to Ollama. */
export interface OllamaRequest {
  prompt: string;
  model?: string;
  system?: string;
  temperature?: number;
}

/** Parsed response returned from an Ollama generation request. */
export interface OllamaResponse {
  text: string;
  model: string;
  totalDuration: number;
  promptTokens: number;
  responseTokens: number;
}

/**
 * Send a generation request to the local Ollama instance.
 * Automatically optimizes the prompt via the memory layer and records token usage.
 *
 * @param req - Prompt, optional model override, optional system prompt, and temperature.
 * @returns Parsed response including generated text and token counts.
 * @throws If Ollama returns a non-OK HTTP status.
 */
export async function generate(req: OllamaRequest): Promise<OllamaResponse> {
  const prompt = await optimizePrompt(req.prompt);

  const res = await fetch(`${OLLAMA_HOST}/api/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      model: req.model || OLLAMA_MODEL,
      prompt,
      system: req.system,
      stream: false,
      options: {
        num_ctx: OLLAMA_CTX_SIZE,
        temperature: req.temperature ?? 0.3,
      },
    }),
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(`Ollama error ${res.status}: ${body}`);
  }

  const data = await res.json() as {
    response: string;
    model: string;
    total_duration: number;
    prompt_eval_count?: number;
    eval_count?: number;
  };

  const result: OllamaResponse = {
    text: data.response,
    model: data.model,
    totalDuration: data.total_duration,
    promptTokens: data.prompt_eval_count ?? 0,
    responseTokens: data.eval_count ?? 0,
  };

  trackUsage(result.promptTokens, result.responseTokens);
  return result;
}

/**
 * Check whether the local Ollama server is reachable.
 * @returns `true` if Ollama responds to a tags request, `false` otherwise.
 */
export async function isOllamaRunning(): Promise<boolean> {
  try {
    const res = await fetch(`${OLLAMA_HOST}/api/tags`);
    return res.ok;
  } catch {
    return false;
  }
}

/**
 * List the names of all models currently available in Ollama.
 * @returns Array of model name strings, or an empty array if the request fails.
 */
export async function listModels(): Promise<string[]> {
  const res = await fetch(`${OLLAMA_HOST}/api/tags`);
  if (!res.ok) return [];
  const data = await res.json() as { models: { name: string }[] };
  return data.models.map(m => m.name);
}
