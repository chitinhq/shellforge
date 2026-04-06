package normalizer

import (
	"strings"
	"testing"

	"github.com/chitinhq/shellforge/internal/action"
)

// TestClassifyShellRisk_ReadOnly verifies that read-only commands are correctly
// classified using word-boundary matching (not prefix matching).
func TestClassifyShellRisk_ReadOnly(t *testing.T) {
	cases := []struct {
		command string
		want    action.RiskLevel
	}{
		// Exact matches
		{"ls", action.RiskReadOnly},
		{"cat", action.RiskReadOnly},
		{"grep", action.RiskReadOnly},
		{"find", action.RiskReadOnly},
		{"go test", action.RiskReadOnly},
		// Space-bounded prefix — still read-only
		{"ls -la /tmp", action.RiskReadOnly},
		{"cat /etc/passwd", action.RiskReadOnly},
		{"grep -r pattern src/", action.RiskReadOnly},
		{"find . -name '*.go'", action.RiskReadOnly},
		{"go test ./...", action.RiskReadOnly},
		{"go vet ./...", action.RiskReadOnly},
		// Regression: prefix false-positives from issue #63
		{"catalog_tool", action.RiskMutating},  // was wrongly read-only before fix
		{"finder.sh", action.RiskMutating},      // was wrongly read-only before fix
		{"catapult deploy", action.RiskMutating}, // starts with "cat" but not "cat "
		{"echo_service start", action.RiskMutating},
		{"grep_wrapper -v foo", action.RiskMutating},
	}

	for _, tc := range cases {
		got := classifyShellRisk(tc.command)
		if got != tc.want {
			t.Errorf("classifyShellRisk(%q) = %q, want %q", tc.command, got, tc.want)
		}
	}
}

// TestClassifyShellRisk_Destructive verifies destructive pattern detection.
func TestClassifyShellRisk_Destructive(t *testing.T) {
	cases := []string{
		"rm file.txt",
		"rm -rf /tmp/work",
		"git push origin main",
		"git reset --hard HEAD~1",
		"chmod 777 /etc/passwd",
		"chown root:root /etc",
		"kill -9 1234",
		"dd if=/dev/zero of=/dev/sda",
	}

	for _, cmd := range cases {
		got := classifyShellRisk(cmd)
		if got != action.RiskDestructive {
			t.Errorf("classifyShellRisk(%q) = %q, want %q", cmd, got, action.RiskDestructive)
		}
	}
}

// TestClassifyShellRisk_Mutating verifies the default-mutating fallback.
func TestClassifyShellRisk_Mutating(t *testing.T) {
	cases := []string{
		"go build ./...",
		"make install",
		"curl https://example.com",
		"python script.py",
		"npm install",
		"docker run ubuntu",
	}

	for _, cmd := range cases {
		got := classifyShellRisk(cmd)
		if got != action.RiskMutating {
			t.Errorf("classifyShellRisk(%q) = %q, want %q", cmd, got, action.RiskMutating)
		}
	}
}

// TestClassifyTool verifies tool-name-to-ActionType mapping.
func TestClassifyTool(t *testing.T) {
	cases := []struct {
		tool       string
		params     map[string]string
		wantType   action.ActionType
		wantRisk   action.RiskLevel
		wantScope  action.Scope
	}{
		{"read_file", nil, action.FileRead, action.RiskReadOnly, action.ScopeFile},
		{"write_file", nil, action.FileWrite, action.RiskMutating, action.ScopeFile},
		{"list_files", nil, action.FileRead, action.RiskReadOnly, action.ScopeDirectory},
		{"search_files", nil, action.FileRead, action.RiskReadOnly, action.ScopeDirectory},
		{"run_shell", map[string]string{"command": "ls -la"}, action.ShellExec, action.RiskReadOnly, action.ScopeSystem},
		{"run_shell", map[string]string{"command": "rm -rf /"}, action.ShellExec, action.RiskDestructive, action.ScopeSystem},
		{"unknown_tool", nil, action.ShellExec, action.RiskMutating, action.ScopeSystem},
	}

	for _, tc := range cases {
		gotType, gotRisk, gotScope := classifyTool(tc.tool, tc.params)
		if gotType != tc.wantType || gotRisk != tc.wantRisk || gotScope != tc.wantScope {
			t.Errorf("classifyTool(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.tool, gotType, gotRisk, gotScope,
				tc.wantType, tc.wantRisk, tc.wantScope)
		}
	}
}

// TestNormalize verifies the Normalize function produces well-formed Proposals.
func TestNormalize(t *testing.T) {
	p := Normalize("run-1", 3, "qa-agent", "read_file", map[string]string{"path": "/etc/hosts"})

	if p.ID != "run-1_3" {
		t.Errorf("ID = %q, want %q", p.ID, "run-1_3")
	}
	if p.RunID != "run-1" {
		t.Errorf("RunID = %q, want %q", p.RunID, "run-1")
	}
	if p.Sequence != 3 {
		t.Errorf("Sequence = %d, want 3", p.Sequence)
	}
	if p.Agent != "qa-agent" {
		t.Errorf("Agent = %q, want %q", p.Agent, "qa-agent")
	}
	if p.Type != action.FileRead {
		t.Errorf("Type = %q, want %q", p.Type, action.FileRead)
	}
	if p.Target != "/etc/hosts" {
		t.Errorf("Target = %q, want %q", p.Target, "/etc/hosts")
	}
	if p.Risk != action.RiskReadOnly {
		t.Errorf("Risk = %q, want %q", p.Risk, action.RiskReadOnly)
	}
}

// TestFingerprint verifies deterministic, content-addressed fingerprinting.
func TestFingerprint(t *testing.T) {
	p1 := Normalize("run-1", 1, "agent-a", "write_file", map[string]string{"path": "foo.go", "content": "hello"})
	p2 := Normalize("run-2", 5, "agent-b", "write_file", map[string]string{"path": "foo.go", "content": "hello"})
	p3 := Normalize("run-1", 1, "agent-a", "write_file", map[string]string{"path": "bar.go", "content": "hello"})

	fp1 := Fingerprint(p1)
	fp2 := Fingerprint(p2)
	fp3 := Fingerprint(p3)

	// Same type+target+params = same fingerprint (run/sequence/agent don't affect it)
	if fp1 != fp2 {
		t.Errorf("identical content proposals should have same fingerprint: %q vs %q", fp1, fp2)
	}
	// Different target = different fingerprint
	if fp1 == fp3 {
		t.Errorf("different target proposals should have different fingerprints")
	}
	// Fingerprint should be 16 hex chars
	if len(fp1) != 16 {
		t.Errorf("fingerprint length = %d, want 16", len(fp1))
	}
	// Should be hex
	for _, c := range fp1 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("fingerprint %q contains non-hex character %q", fp1, c)
		}
	}
}
