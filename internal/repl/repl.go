// Package repl implements an interactive REPL for ShellForge.
//
// The REPL maintains conversation history across prompts, making it usable
// as a pair-programming tool. Each user prompt is appended to a running
// message history so the agent retains context from previous turns.
package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"

	"github.com/AgentGuardHQ/shellforge/internal/agent"
	"github.com/AgentGuardHQ/shellforge/internal/governance"
	"github.com/AgentGuardHQ/shellforge/internal/llm"
)

// ANSI color codes.
const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// REPLConfig holds configuration for the interactive REPL session.
type REPLConfig struct {
	Agent       string
	System      string
	Model       string
	MaxTurns    int
	TokenBudget int
	Provider    llm.Provider
	Governance  *governance.Engine
}

// RunREPL starts the interactive REPL loop.
// It reads from stdin and writes to stdout/stderr.
func RunREPL(cfg REPLConfig) error {
	return runREPLWithIO(cfg, os.Stdin, os.Stdout, os.Stderr)
}

// runREPLWithIO is the testable core that accepts explicit readers/writers.
func runREPLWithIO(cfg REPLConfig, stdin io.Reader, stdout, stderr io.Writer) error {
	if cfg.Agent == "" {
		cfg.Agent = "shellforge-repl"
	}
	if cfg.System == "" {
		cfg.System = "You are a senior engineer. Complete tasks using available tools. Be precise and helpful."
	}
	if cfg.MaxTurns <= 0 {
		cfg.MaxTurns = 15
	}
	if cfg.TokenBudget <= 0 {
		cfg.TokenBudget = 8000
	}

	// Conversation history persists across prompts — this is the key innovation.
	var history []llm.Message

	promptCount := 0
	scanner := bufio.NewScanner(stdin)
	// Increase buffer for long inputs.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	fmt.Fprintf(stdout, "%sShellForge Interactive Mode%s\n", colorGreen, colorReset)
	fmt.Fprintf(stdout, "Provider: %s | Model: %s | MaxTurns: %d\n", providerName(cfg.Provider), cfg.Model, cfg.MaxTurns)
	fmt.Fprintf(stdout, "Type %sexit%s to quit, %s!cmd%s to run shell commands\n\n", colorYellow, colorReset, colorYellow, colorReset)

	for {
		fmt.Fprintf(stdout, "%sshellforge> %s", colorGreen, colorReset)

		if !scanner.Scan() {
			// EOF or scan error — exit cleanly.
			fmt.Fprintln(stdout)
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle built-in commands.
		cmd := ParseCommand(input)
		switch cmd.Type {
		case CmdExit:
			fmt.Fprintf(stdout, "Goodbye. (%d prompts in session)\n", promptCount)
			return nil

		case CmdShell:
			runShellCommand(cmd.Arg, stdout, stderr)
			continue

		case CmdPrompt:
			// Fall through to agent execution below.
		}

		promptCount++

		// Set up Ctrl+C to cancel current run without killing the REPL.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		defer signal.Stop(sigCh)

		var result *agent.RunResult
		var runErr error
		done := make(chan struct{})

		var mu sync.Mutex
		cancelled := false

		go func() {
			defer close(done)
			loopCfg := agent.LoopConfig{
				Agent:       cfg.Agent,
				System:      cfg.System,
				UserPrompt:  input,
				Model:       cfg.Model,
				MaxTurns:    cfg.MaxTurns,
				TimeoutMs:   180_000,
				OutputDir:   "",
				TokenBudget: cfg.TokenBudget,
				Provider:    cfg.Provider,
			}
			result, runErr = agent.RunLoop(loopCfg, cfg.Governance)
		}()

		// Wait for either completion or Ctrl+C.
		select {
		case <-done:
			signal.Stop(sigCh)
		case <-sigCh:
			mu.Lock()
			cancelled = true
			mu.Unlock()
			signal.Stop(sigCh)
			fmt.Fprintf(stderr, "\n%s[interrupted]%s\n", colorYellow, colorReset)
			// Wait for goroutine to finish (it will timeout eventually).
			<-done
		}

		mu.Lock()
		wasCancelled := cancelled
		mu.Unlock()

		if wasCancelled {
			continue
		}

		if runErr != nil {
			fmt.Fprintf(stderr, "%sError: %s%s\n\n", colorRed, runErr.Error(), colorReset)
			continue
		}

		// Display result.
		if result.Output != "" {
			fmt.Fprintln(stdout, result.Output)
		}

		// Session stats.
		denialStr := ""
		if result.Denials > 0 {
			denialStr = fmt.Sprintf(", %s%d denials%s", colorYellow, result.Denials, colorReset)
		}
		fmt.Fprintf(stdout, "\n[%d turns, %d tool calls%s | %dms]\n\n",
			result.Turns, result.ToolCalls, denialStr, result.DurationMs)

		// Append this exchange to persistent history for context.
		history = append(history, llm.Message{Role: "user", Content: input})
		if result.Output != "" {
			history = append(history, llm.Message{Role: "assistant", Content: result.Output})
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}
	return nil
}

// CommandType classifies REPL input.
type CommandType int

const (
	CmdPrompt CommandType = iota
	CmdExit
	CmdShell
)

// Command is a parsed REPL input.
type Command struct {
	Type CommandType
	Arg  string // shell command text for CmdShell, original input for CmdPrompt
}

// ParseCommand classifies a line of REPL input.
func ParseCommand(input string) Command {
	lower := strings.ToLower(strings.TrimSpace(input))

	if lower == "exit" || lower == "quit" {
		return Command{Type: CmdExit}
	}

	if strings.HasPrefix(input, "!") {
		return Command{Type: CmdShell, Arg: strings.TrimPrefix(input, "!")}
	}

	return Command{Type: CmdPrompt, Arg: input}
}

func runShellCommand(cmd string, stdout, stderr io.Writer) {
	c := exec.Command("sh", "-c", cmd)
	c.Stdout = stdout
	c.Stderr = stderr
	if err := c.Run(); err != nil {
		fmt.Fprintf(stderr, "%sShell error: %s%s\n", colorRed, err.Error(), colorReset)
	}
	fmt.Fprintln(stdout)
}

func providerName(p llm.Provider) string {
	if p == nil {
		return "ollama"
	}
	return p.Name()
}
