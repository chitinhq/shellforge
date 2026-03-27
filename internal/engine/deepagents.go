package engine

import (
"fmt"
"os"
"os/exec"
"strings"
"time"
)

// DeepAgentsEngine wraps the DeepAgents framework (LangChain/LangGraph).
// DeepAgents provides multi-step planning and autonomous task decomposition.
// Available as npm package (deepagents@1.8.6) or Python (pip install deepagents).
type DeepAgentsEngine struct{}

func (e *DeepAgentsEngine) Name() string { return "deepagents" }

func (e *DeepAgentsEngine) Available() bool {
// Check for Node.js module
cmd := exec.Command("node", "-e", "require('deepagents')")
if cmd.Run() == nil {
return true
}
// Check for Python module
cmd = exec.Command("python3", "-c", "import deepagents")
return cmd.Run() == nil
}

func (e *DeepAgentsEngine) Run(task Task) (*Result, error) {
start := time.Now()

if !e.Available() {
return nil, fmt.Errorf("deepagents not installed. Install: npm i deepagents OR pip install deepagents")
}

// DeepAgents Node.js invocation with governance integration
script := fmt.Sprintf(`
const { createDeepAgent } = require('deepagents');
const agent = createDeepAgent({
  model: '%s',
  maxSteps: %d,
  timeout: %d * 1000,
  governance: {
    policyFile: 'agentguard.yaml',
    mode: 'enforce',
    onToolCall: (tool, params) => {
      console.error('[🛡️ AgentGuard] evaluating: ' + tool);
    }
  }
});
agent.invoke({
  messages: [{ role: 'user', content: %q }]
}).then(r => {
  console.log(JSON.stringify({
    output: r.output || r.messages?.slice(-1)[0]?.content || '',
    turns: r.steps || 0,
    toolCalls: r.toolCalls || 0,
    tokens: { prompt: r.promptTokens || 0, response: r.responseTokens || 0 }
  }));
}).catch(e => {
  console.error('DeepAgents error:', e.message);
  process.exit(1);
});
`, task.Model, task.MaxTurns, task.Timeout, task.Prompt)

cmd := exec.Command("node", "-e", script)
cmd.Dir = task.WorkDir
cmd.Env = append(os.Environ(),
"AGENTGUARD_POLICY=agentguard.yaml",
)

out, err := cmd.CombinedOutput()
output := strings.TrimSpace(string(out))
duration := time.Since(start).Milliseconds()

if err != nil && output == "" {
return nil, fmt.Errorf("deepagents failed: %w", err)
}

return &Result{
Success:    err == nil,
Output:     output,
DurationMs: duration,
}, nil
}
