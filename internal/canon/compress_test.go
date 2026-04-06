package canon

import (
	"strings"
	"testing"
)

func TestRTKCommand_GitStatus(t *testing.T) {
	args, ok := RTKCommand("git status")
	if !ok {
		t.Fatal("expected RTK match for git status")
	}
	if len(args) < 2 || args[0] != "git" || args[1] != "status" {
		t.Errorf("args=%v, want [git status]", args)
	}
}

func TestRTKCommand_CatFile(t *testing.T) {
	args, ok := RTKCommand("cat foo.txt")
	if !ok {
		t.Fatal("expected RTK match for cat")
	}
	if len(args) < 2 || args[0] != "read" || args[1] != "foo.txt" {
		t.Errorf("args=%v, want [read foo.txt]", args)
	}
}

func TestRTKCommand_RgGrep(t *testing.T) {
	args, ok := RTKCommand("rg -n pattern src/")
	if !ok {
		t.Fatal("expected RTK match for rg")
	}
	if args[0] != "grep" {
		t.Errorf("args[0]=%q, want 'grep'", args[0])
	}
}

func TestRTKCommand_Chain(t *testing.T) {
	_, ok := RTKCommand("git add . && git commit -m test")
	if ok {
		t.Error("chains should not produce a specific RTK command")
	}
}

func TestRTKCommand_Unknown(t *testing.T) {
	_, ok := RTKCommand("my-special-tool --flag")
	if ok {
		t.Error("unknown tools should not match")
	}
}

func TestCompressOutput_GitLog(t *testing.T) {
	// Generate 100-line git log output.
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "abc1234 commit message "+itoa(i))
	}
	output := strings.Join(lines, "\n")

	cmd := Command{Tool: "git", Action: "log"}
	compressed := CompressOutput(cmd, output)

	// Should be shorter than original.
	if len(compressed) >= len(output) {
		t.Errorf("compression didn't reduce: %d >= %d", len(compressed), len(output))
	}
	// Should contain "more lines" indicator.
	if !strings.Contains(compressed, "more lines") {
		t.Error("compressed output should indicate truncation")
	}
}

func TestCompressOutput_GrepOutput(t *testing.T) {
	// Generate grep output with many matches per file — enough to trigger compression.
	var lines []string
	for i := 0; i < 80; i++ {
		lines = append(lines, "src/main.go:"+itoa(i)+": match here")
	}
	output := strings.Join(lines, "\n")

	cmd := Command{Tool: "grep"}
	compressed := CompressOutput(cmd, output)

	// Should keep first 5 + truncation indicator — well under 80 lines.
	compLines := strings.Split(compressed, "\n")
	if len(compLines) >= 80 {
		t.Errorf("grep should compress, got %d lines (same as input)", len(compLines))
	}
	if !strings.Contains(compressed, "more matches") {
		t.Error("should indicate truncated matches")
	}
}

func TestCompressOutput_ShortOutput(t *testing.T) {
	cmd := Command{Tool: "ls"}
	short := "file1.txt\nfile2.txt\n"
	result := CompressOutput(cmd, short)
	if result != short {
		t.Error("short output should pass through unchanged")
	}
}

func TestCompressOutput_ReadFile(t *testing.T) {
	// Generate 100-line file output.
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "line "+itoa(i))
	}
	output := strings.Join(lines, "\n")

	cmd := Command{Tool: "read"}
	compressed := CompressOutput(cmd, output)

	// Should keep head + tail.
	if !strings.Contains(compressed, "line 0") {
		t.Error("should keep first lines")
	}
	if !strings.Contains(compressed, "line 99") {
		t.Error("should keep last lines")
	}
	if !strings.Contains(compressed, "lines omitted") {
		t.Error("should indicate omission")
	}
}
