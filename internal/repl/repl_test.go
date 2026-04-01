package repl

import (
	"bytes"
	"strings"
	"testing"
)

// ── ParseCommand tests ──

func TestParseCommand_Exit(t *testing.T) {
	cases := []string{"exit", "Exit", "EXIT", "quit", "Quit", "QUIT"}
	for _, input := range cases {
		cmd := ParseCommand(input)
		if cmd.Type != CmdExit {
			t.Errorf("ParseCommand(%q) = %d, want CmdExit", input, cmd.Type)
		}
	}
}

func TestParseCommand_Shell(t *testing.T) {
	cases := []struct {
		input string
		arg   string
	}{
		{"!ls", "ls"},
		{"!git status", "git status"},
		{"!echo hello world", "echo hello world"},
		{"!", ""},
	}
	for _, tc := range cases {
		cmd := ParseCommand(tc.input)
		if cmd.Type != CmdShell {
			t.Errorf("ParseCommand(%q).Type = %d, want CmdShell", tc.input, cmd.Type)
		}
		if cmd.Arg != tc.arg {
			t.Errorf("ParseCommand(%q).Arg = %q, want %q", tc.input, cmd.Arg, tc.arg)
		}
	}
}

func TestParseCommand_Prompt(t *testing.T) {
	cases := []string{
		"read the README",
		"what files are here?",
		"exit now please", // not exactly "exit"
		"! leading space",  // "!" at start, this IS shell (! + " leading space")
		"review this code",
	}
	for _, input := range cases {
		cmd := ParseCommand(input)
		// "! leading space" starts with "!" so it is a shell command
		if strings.HasPrefix(input, "!") {
			if cmd.Type != CmdShell {
				t.Errorf("ParseCommand(%q).Type = %d, want CmdShell", input, cmd.Type)
			}
		} else {
			if cmd.Type != CmdPrompt {
				t.Errorf("ParseCommand(%q).Type = %d, want CmdPrompt", input, cmd.Type)
			}
		}
	}
}

func TestParseCommand_EmptyAndWhitespace(t *testing.T) {
	// These would be caught by the REPL loop (empty check before ParseCommand),
	// but ParseCommand itself treats them as prompts.
	cmd := ParseCommand("  exit  ")
	if cmd.Type != CmdExit {
		t.Errorf("ParseCommand with whitespace around 'exit' should be CmdExit, got %d", cmd.Type)
	}
}

// ── runShellCommand tests ──

func TestRunShellCommand_Success(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runShellCommand("echo hello", &stdout, &stderr)

	if !strings.Contains(stdout.String(), "hello") {
		t.Fatalf("expected 'hello' in stdout, got %q", stdout.String())
	}
	if stderr.Len() > 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
}

func TestRunShellCommand_Failure(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runShellCommand("false", &stdout, &stderr)

	if !strings.Contains(stderr.String(), "Shell error") {
		t.Fatalf("expected error message in stderr, got %q", stderr.String())
	}
}

func TestRunShellCommand_OutputCapture(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runShellCommand("echo line1; echo line2", &stdout, &stderr)

	output := stdout.String()
	if !strings.Contains(output, "line1") || !strings.Contains(output, "line2") {
		t.Fatalf("expected both lines, got %q", output)
	}
}

// ── providerName tests ──

func TestProviderName_Nil(t *testing.T) {
	name := providerName(nil)
	if name != "ollama" {
		t.Fatalf("expected 'ollama' for nil provider, got %q", name)
	}
}

// ── REPLConfig defaults ──

func TestREPLConfig_Defaults(t *testing.T) {
	// Test that runREPLWithIO applies defaults when fields are zero.
	// We send "exit" immediately so it doesn't block.
	stdin := strings.NewReader("exit\n")
	var stdout, stderr bytes.Buffer

	cfg := REPLConfig{
		// Leave everything zero/empty.
	}

	// This will fail because Governance is nil, but that's expected when
	// there's no governance engine — the REPL will print the banner and exit.
	// We just want to verify it doesn't panic on zero config.
	// Note: agent.RunLoop requires governance, so exit before that.
	err := runREPLWithIO(cfg, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "ShellForge Interactive Mode") {
		t.Fatalf("expected banner, got %q", output)
	}
	if !strings.Contains(output, "Goodbye") {
		t.Fatalf("expected goodbye message, got %q", output)
	}
}

func TestREPL_EmptyLines(t *testing.T) {
	stdin := strings.NewReader("\n\n\nexit\n")
	var stdout, stderr bytes.Buffer

	cfg := REPLConfig{}
	err := runREPLWithIO(cfg, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "0 prompts") {
		t.Fatalf("expected 0 prompts (empty lines skipped), got %q", output)
	}
}

func TestREPL_EOF(t *testing.T) {
	// Empty input — immediate EOF.
	stdin := strings.NewReader("")
	var stdout, stderr bytes.Buffer

	cfg := REPLConfig{}
	err := runREPLWithIO(cfg, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestREPL_ShellCommand(t *testing.T) {
	stdin := strings.NewReader("!echo shelltest\nexit\n")
	var stdout, stderr bytes.Buffer

	cfg := REPLConfig{}
	err := runREPLWithIO(cfg, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "shelltest") {
		t.Fatalf("expected shell output 'shelltest', got %q", output)
	}
}

func TestREPL_QuitAlias(t *testing.T) {
	stdin := strings.NewReader("quit\n")
	var stdout, stderr bytes.Buffer

	cfg := REPLConfig{}
	err := runREPLWithIO(cfg, stdin, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Goodbye") {
		t.Fatalf("expected goodbye on quit, got %q", stdout.String())
	}
}
