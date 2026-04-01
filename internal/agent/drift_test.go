package agent

import "testing"

func TestNewDriftDetector(t *testing.T) {
	d := newDriftDetector("fix the auth bug")
	if d.taskSpec != "fix the auth bug" {
		t.Errorf("expected task spec, got %s", d.taskSpec)
	}
	if len(d.actionLog) != 0 {
		t.Error("expected empty action log")
	}
}

func TestDriftDetector_Record(t *testing.T) {
	d := newDriftDetector("task")
	d.record("read_file", map[string]string{"path": "auth.go"})
	d.record("run_shell", map[string]string{"command": "go test"})
	if len(d.actionLog) != 2 {
		t.Errorf("expected 2 actions, got %d", len(d.actionLog))
	}
	if d.actionLog[0] != "read_file → auth.go" {
		t.Errorf("expected 'read_file → auth.go', got %s", d.actionLog[0])
	}
}

func TestDriftDetector_ShouldCheck(t *testing.T) {
	d := newDriftDetector("task")
	if d.shouldCheck(0) {
		t.Error("should not check at 0")
	}
	if d.shouldCheck(3) {
		t.Error("should not check at 3")
	}
	if !d.shouldCheck(5) {
		t.Error("should check at 5")
	}
	if !d.shouldCheck(10) {
		t.Error("should check at 10")
	}
}

func TestParseScore(t *testing.T) {
	cases := []struct {
		input string
		want  int
	}{
		{"8", 8},
		{"  7  ", 7},
		{"Score: 9", 9},
		{"3/10", 3},
		{"no score here", 10}, // default
		{"", 10},
	}
	for _, c := range cases {
		if got := parseScore(c.input); got != c.want {
			t.Errorf("parseScore(%q) = %d, want %d", c.input, got, c.want)
		}
	}
}

func TestDriftDetector_Evaluate_OK(t *testing.T) {
	d := newDriftDetector("task")
	if d.evaluate(8) != driftOK {
		t.Error("score 8 should be OK")
	}
	if d.evaluate(10) != driftOK {
		t.Error("score 10 should be OK")
	}
}

func TestDriftDetector_Evaluate_Warn(t *testing.T) {
	d := newDriftDetector("task")
	if d.evaluate(6) != driftWarn {
		t.Error("score 6 should warn")
	}
	if d.warnings != 1 {
		t.Errorf("expected 1 warning, got %d", d.warnings)
	}
}

func TestDriftDetector_Evaluate_Kill(t *testing.T) {
	d := newDriftDetector("task")
	// First low score → warn
	if d.evaluate(4) != driftWarn {
		t.Error("first score 4 should warn")
	}
	// Second consecutive low score → kill
	if d.evaluate(3) != driftKill {
		t.Error("second low score should kill")
	}
}

func TestDriftDetector_Evaluate_ResetOnGoodScore(t *testing.T) {
	d := newDriftDetector("task")
	d.evaluate(4) // low → warn
	d.evaluate(8) // good → reset
	// Next low should warn again, not kill
	if d.evaluate(3) != driftWarn {
		t.Error("after reset, first low should warn not kill")
	}
}
