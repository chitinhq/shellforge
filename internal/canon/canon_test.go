package canon

import (
	"encoding/json"
	"testing"
)

func TestReadEquivalence(t *testing.T) {
	// cat, head, and tail with no special flags should all canonicalize to the same "read" tool.
	cat := ParseOne("cat foo.txt")
	head := ParseOne("head foo.txt")
	tail := ParseOne("tail foo.txt")

	if cat.Tool != "read" {
		t.Errorf("cat → tool=%q, want 'read'", cat.Tool)
	}
	if head.Tool != "read" {
		t.Errorf("head → tool=%q, want 'read'", head.Tool)
	}
	if tail.Tool != "read" {
		t.Errorf("tail → tool=%q, want 'read'", tail.Tool)
	}

	// Without flags they should have the same args and thus same digest.
	if cat.Digest != head.Digest {
		t.Errorf("cat and head should have same digest: %s vs %s", cat.Digest, head.Digest)
	}
	if cat.Digest != tail.Digest {
		t.Errorf("cat and tail should have same digest: %s vs %s", cat.Digest, tail.Digest)
	}
}

func TestHeadTailWithFlags(t *testing.T) {
	// head -20 foo.txt and tail -20 foo.txt should differ (head-lines vs tail-lines).
	h := ParseOne("head -n 20 foo.txt")
	ta := ParseOne("tail -n 20 foo.txt")

	if h.Digest == ta.Digest {
		t.Error("head -n 20 and tail -n 20 should have different digests")
	}
	if h.Flags["head-lines"] != "20" {
		t.Errorf("head flags=%v, want head-lines=20", h.Flags)
	}
	if ta.Flags["tail-lines"] != "20" {
		t.Errorf("tail flags=%v, want tail-lines=20", ta.Flags)
	}
}

func TestGrepEquivalence(t *testing.T) {
	// grep -rn pattern . and rg -n pattern . should produce the same digest.
	// rg is recursive by default, so -r is stripped. grep -r normalizes to recursive.
	g := ParseOne("grep -rn pattern .")
	r := ParseOne("rg -n pattern .")

	if g.Tool != "grep" || r.Tool != "grep" {
		t.Errorf("tools: grep=%q, rg=%q", g.Tool, r.Tool)
	}
	if g.Digest != r.Digest {
		t.Errorf("grep -rn and rg -n should have same digest: %s vs %s\ngrep flags=%v args=%v\nrg flags=%v args=%v",
			g.Digest, r.Digest, g.Flags, g.Args, r.Flags, r.Args)
	}
}

func TestGitLogOneline(t *testing.T) {
	// git log --oneline and git log --pretty=oneline should produce the same digest.
	a := ParseOne("git log --oneline")
	b := ParseOne("git log --pretty=oneline")

	if a.Tool != "git" || a.Action != "log" {
		t.Errorf("git log: tool=%q action=%q", a.Tool, a.Action)
	}
	if a.Digest != b.Digest {
		t.Errorf("--oneline and --pretty=oneline should match: %s vs %s\na flags=%v\nb flags=%v",
			a.Digest, b.Digest, a.Flags, b.Flags)
	}
}

func TestChainParsing(t *testing.T) {
	p := Parse("git add . && git commit -m 'fix bug'")

	if len(p.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(p.Segments))
	}
	if p.Segments[0].Op != OpNone {
		t.Errorf("first segment op=%q, want empty", p.Segments[0].Op)
	}
	if p.Segments[0].Command.Tool != "git" || p.Segments[0].Command.Action != "add" {
		t.Errorf("first segment: tool=%q action=%q", p.Segments[0].Command.Tool, p.Segments[0].Command.Action)
	}
	if p.Segments[1].Op != OpAnd {
		t.Errorf("second segment op=%q, want &&", p.Segments[1].Op)
	}
	if p.Segments[1].Command.Tool != "git" || p.Segments[1].Command.Action != "commit" {
		t.Errorf("second segment: tool=%q action=%q", p.Segments[1].Command.Tool, p.Segments[1].Command.Action)
	}
}

func TestPipeParsing(t *testing.T) {
	p := Parse("git log | head -20")

	if len(p.Segments) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(p.Segments))
	}
	if p.Segments[1].Op != OpPipe {
		t.Errorf("second segment op=%q, want |", p.Segments[1].Op)
	}
	if p.Segments[1].Command.Tool != "read" {
		t.Errorf("head piped: tool=%q, want 'read'", p.Segments[1].Command.Tool)
	}
}

func TestEnvVarPrefix(t *testing.T) {
	cmd := ParseOne("DEEPSEEK_API_KEY=sk-1234 python3 main.py")

	if cmd.Tool != "python" {
		t.Errorf("tool=%q, want 'python'", cmd.Tool)
	}
	if len(cmd.Args) < 1 || cmd.Args[0] != "main.py" {
		t.Errorf("args=%v, want [main.py]", cmd.Args)
	}
}

func TestSensitiveMasking(t *testing.T) {
	cmd := ParseOne("curl -H 'Authorization: Bearer sk-ant-api03-longtoken12345678901234567890' http://example.com")

	for _, arg := range cmd.Args {
		if len(arg) > 30 {
			t.Errorf("sensitive value not masked: %s", arg)
		}
	}
}

func TestQuotedArgs(t *testing.T) {
	cmd := ParseOne(`git commit -m "this is a message"`)

	if cmd.Tool != "git" || cmd.Action != "commit" {
		t.Errorf("tool=%q action=%q", cmd.Tool, cmd.Action)
	}
	// -m takes a value, should be in flags
	if cmd.Flags["m"] != "this is a message" {
		t.Errorf("flags=%v, want m='this is a message'", cmd.Flags)
	}
}

func TestEmptyCommand(t *testing.T) {
	cmd := ParseOne("")
	if cmd.Tool != "unknown" {
		t.Errorf("empty command tool=%q, want 'unknown'", cmd.Tool)
	}
}

func TestUnknownTool(t *testing.T) {
	cmd := ParseOne("my-custom-tool --flag value")
	if cmd.Tool != "my-custom-tool" {
		t.Errorf("unknown tool=%q, want 'my-custom-tool'", cmd.Tool)
	}
}

func TestJSONOutput(t *testing.T) {
	cmd := ParseOne("git status")
	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	var back Command
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if back.Tool != "git" || back.Action != "status" || back.Digest != cmd.Digest {
		t.Errorf("roundtrip failed: %+v", back)
	}
}

func TestDigestDeterminism(t *testing.T) {
	// Same command parsed twice should always produce the same digest.
	a := ParseOne("docker ps -a --format '{{.Names}}'")
	b := ParseOne("docker ps -a --format '{{.Names}}'")

	if a.Digest != b.Digest {
		t.Errorf("determinism failed: %s vs %s", a.Digest, b.Digest)
	}
}

func TestComplexChain(t *testing.T) {
	p := Parse("cd /tmp && ls -la && cat foo.txt | grep pattern")

	if len(p.Segments) != 4 {
		t.Fatalf("expected 4 segments, got %d", len(p.Segments))
	}
	if p.Segments[0].Command.Tool != "cd" {
		t.Errorf("seg0 tool=%q", p.Segments[0].Command.Tool)
	}
	if p.Segments[1].Op != OpAnd || p.Segments[1].Command.Tool != "ls" {
		t.Errorf("seg1 op=%q tool=%q", p.Segments[1].Op, p.Segments[1].Command.Tool)
	}
	if p.Segments[2].Op != OpAnd || p.Segments[2].Command.Tool != "read" {
		t.Errorf("seg2 op=%q tool=%q", p.Segments[2].Op, p.Segments[2].Command.Tool)
	}
	if p.Segments[3].Op != OpPipe || p.Segments[3].Command.Tool != "grep" {
		t.Errorf("seg3 op=%q tool=%q", p.Segments[3].Op, p.Segments[3].Command.Tool)
	}
}
