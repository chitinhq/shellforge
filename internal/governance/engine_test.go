package governance

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConfig writes a temporary chitin.yaml and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "chitin.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}
	return path
}

const enforceConfig = `
mode: enforce
policies:
  - name: no-destructive-rm
    description: Block rm commands
    match:
      command: rm
    action: deny
    message: rm is not allowed in enforce mode
  - name: no-git-push
    description: Block git push
    match:
      command: git
      args_contain: ["push"]
    action: deny
    message: git push is not allowed
  - name: monitor-writes
    description: Log all writes
    match:
      command: write_file
    action: monitor
    message: write observed
`

const monitorConfig = `
mode: monitor
policies:
  - name: no-destructive-rm
    match:
      command: rm
    action: deny
    message: rm is not allowed
`

const timeoutConfig = `
mode: enforce
policies:
  - name: long-running-budget
    match:
      command: "*"
    action: monitor
    message: budget policy
    timeout_seconds: 600
`

// TestEvaluate_EnforceDeny verifies that deny policies block execution in enforce mode.
func TestEvaluate_EnforceDeny(t *testing.T) {
	path := writeConfig(t, enforceConfig)
	eng, err := NewEngine(path)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}

	d := eng.Evaluate("run_shell", map[string]string{"command": "rm -rf /tmp/work"})
	if d.Allowed {
		t.Error("rm should be denied in enforce mode")
	}
	if d.PolicyName != "no-destructive-rm" {
		t.Errorf("PolicyName = %q, want %q", d.PolicyName, "no-destructive-rm")
	}
	if d.Mode != "enforce" {
		t.Errorf("Mode = %q, want %q", d.Mode, "enforce")
	}
}

// TestEvaluate_EnforceDeny_ArgsContain verifies args_contain matching.
func TestEvaluate_EnforceDeny_ArgsContain(t *testing.T) {
	path := writeConfig(t, enforceConfig)
	eng, _ := NewEngine(path)

	d := eng.Evaluate("run_shell", map[string]string{"command": "git push origin main"})
	if d.Allowed {
		t.Error("git push should be denied in enforce mode")
	}
	if d.PolicyName != "no-git-push" {
		t.Errorf("PolicyName = %q, want %q", d.PolicyName, "no-git-push")
	}

	// git pull should not match the git push policy
	d2 := eng.Evaluate("run_shell", map[string]string{"command": "git pull origin main"})
	if !d2.Allowed {
		t.Error("git pull should be allowed (only push is denied)")
	}
}

// TestEvaluate_MonitorAllow verifies that deny policies only log in monitor mode.
func TestEvaluate_MonitorAllow(t *testing.T) {
	path := writeConfig(t, monitorConfig)
	eng, _ := NewEngine(path)

	d := eng.Evaluate("run_shell", map[string]string{"command": "rm -rf /tmp"})
	if !d.Allowed {
		t.Error("rm should be allowed in monitor mode (deny = log only)")
	}
	if d.PolicyName != "no-destructive-rm" {
		t.Errorf("PolicyName = %q, want %q", d.PolicyName, "no-destructive-rm")
	}
	if d.Mode != "monitor" {
		t.Errorf("Mode = %q, want %q", d.Mode, "monitor")
	}
}

// TestEvaluate_MonitorAction verifies monitor-action policies always allow.
func TestEvaluate_MonitorAction(t *testing.T) {
	path := writeConfig(t, enforceConfig)
	eng, _ := NewEngine(path)

	// monitor-writes policy matches write_file and should always allow
	d := eng.Evaluate("write_file", map[string]string{"command": "write_file", "path": "foo.go"})
	if !d.Allowed {
		t.Error("monitor policy should always allow")
	}
}

// TestEvaluate_DefaultAllow verifies that unmatched commands are allowed.
func TestEvaluate_DefaultAllow(t *testing.T) {
	path := writeConfig(t, enforceConfig)
	eng, _ := NewEngine(path)

	d := eng.Evaluate("run_shell", map[string]string{"command": "go test ./..."})
	if !d.Allowed {
		t.Errorf("go test should be default-allowed, got reason: %q", d.Reason)
	}
	if d.PolicyName != "default-allow" {
		t.Errorf("PolicyName = %q, want %q", d.PolicyName, "default-allow")
	}
}

// TestGetTimeout_PolicyTimeout verifies policy-level timeout is respected.
func TestGetTimeout_PolicyTimeout(t *testing.T) {
	path := writeConfig(t, timeoutConfig)
	eng, _ := NewEngine(path)

	got := eng.GetTimeout()
	if got != 600 {
		t.Errorf("GetTimeout() = %d, want 600", got)
	}
}

// TestGetTimeout_Default verifies the 300s default when no policy sets a timeout.
func TestGetTimeout_Default(t *testing.T) {
	path := writeConfig(t, enforceConfig)
	eng, _ := NewEngine(path)

	got := eng.GetTimeout()
	if got != 300 {
		t.Errorf("GetTimeout() = %d, want 300 (default)", got)
	}
}

// TestNewEngine_DefaultMonitorMode verifies that missing mode defaults to monitor.
func TestNewEngine_DefaultMonitorMode(t *testing.T) {
	cfg := `
policies:
  - name: test
    match:
      command: rm
    action: deny
    message: denied
`
	path := writeConfig(t, cfg)
	eng, err := NewEngine(path)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	if eng.Mode != "monitor" {
		t.Errorf("Mode = %q, want %q (default)", eng.Mode, "monitor")
	}
}

// TestNewEngine_MissingFile verifies error on missing config.
func TestNewEngine_MissingFile(t *testing.T) {
	_, err := NewEngine("/no/such/file.yaml")
	if err == nil {
		t.Error("expected error for missing config file, got nil")
	}
}

// TestNewEngine_InvalidYAML verifies error on malformed config.
func TestNewEngine_InvalidYAML(t *testing.T) {
	path := writeConfig(t, "mode: [\ninvalid yaml")
	_, err := NewEngine(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
