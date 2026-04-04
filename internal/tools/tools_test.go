package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── edit_file tests ──

func TestEditFile_BasicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	r := editFile(map[string]string{
		"path":     path,
		"old_text": "world",
		"new_text": "go",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello go" {
		t.Fatalf("expected 'hello go', got %q", string(data))
	}
}

func TestEditFile_NoMatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello world"), 0o644)

	r := editFile(map[string]string{
		"path":     path,
		"old_text": "foobar",
		"new_text": "baz",
	}, 0)

	if r.Success {
		t.Fatal("expected failure for no match")
	}
	if r.Error != "no_match" {
		t.Fatalf("expected error 'no_match', got %q", r.Error)
	}
}

func TestEditFile_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("aaa bbb aaa"), 0o644)

	r := editFile(map[string]string{
		"path":     path,
		"old_text": "aaa",
		"new_text": "ccc",
	}, 0)

	if r.Success {
		t.Fatal("expected failure for multiple matches")
	}
	if r.Error != "multiple_matches" {
		t.Fatalf("expected error 'multiple_matches', got %q", r.Error)
	}

	// Verify file is unchanged
	data, _ := os.ReadFile(path)
	if string(data) != "aaa bbb aaa" {
		t.Fatalf("file should be unchanged, got %q", string(data))
	}
}

func TestEditFile_FileNotFound(t *testing.T) {
	r := editFile(map[string]string{
		"path":     "/nonexistent/file.txt",
		"old_text": "foo",
		"new_text": "bar",
	}, 0)

	if r.Success {
		t.Fatal("expected failure for missing file")
	}
	if r.Error != "not_found" {
		t.Fatalf("expected error 'not_found', got %q", r.Error)
	}
}

func TestEditFile_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	os.WriteFile(path, []byte("#!/bin/bash\necho hello"), 0o755)

	r := editFile(map[string]string{
		"path":     path,
		"old_text": "echo hello",
		"new_text": "echo goodbye",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	info, _ := os.Stat(path)
	perm := info.Mode().Perm()
	if perm != 0o755 {
		t.Fatalf("expected permissions 0755, got %o", perm)
	}
}

func TestEditFile_MissingParams(t *testing.T) {
	r := editFile(map[string]string{
		"old_text": "foo",
		"new_text": "bar",
	}, 0)
	if r.Success {
		t.Fatal("expected failure for missing path")
	}

	r = editFile(map[string]string{
		"path":     "/tmp/test.txt",
		"new_text": "bar",
	}, 0)
	if r.Success {
		t.Fatal("expected failure for missing old_text")
	}
}

func TestEditFile_MultilineContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := "package main\n\nfunc old() {\n\treturn\n}\n"
	os.WriteFile(path, []byte(content), 0o644)

	r := editFile(map[string]string{
		"path":     path,
		"old_text": "func old() {\n\treturn\n}",
		"new_text": "func newFunc() {\n\tfmt.Println(\"new\")\n}",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "func newFunc()") {
		t.Fatalf("expected new function name, got %q", string(data))
	}
}

func TestEditFile_EmptyNewText(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("keep this remove this keep that"), 0o644)

	r := editFile(map[string]string{
		"path":     path,
		"old_text": " remove this",
		"new_text": "",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "keep this keep that" {
		t.Fatalf("expected deletion, got %q", string(data))
	}
}

// ── glob tests ──

func TestGlob_SimplePattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.go"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0o644)

	r := globFiles(map[string]string{
		"pattern":   "*.go",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	lines := strings.Split(r.Output, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 matches, got %d: %s", len(lines), r.Output)
	}
}

func TestGlob_RecursivePattern(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(sub, "b.go"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(sub, "c.txt"), []byte(""), 0o644)

	r := globFiles(map[string]string{
		"pattern":   "**/*.go",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	lines := strings.Split(r.Output, "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 matches, got %d: %s", len(lines), r.Output)
	}
}

func TestGlob_NoMatches(t *testing.T) {
	dir := t.TempDir()

	r := globFiles(map[string]string{
		"pattern":   "*.xyz",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if r.Output != "No files matched" {
		t.Fatalf("expected 'No files matched', got %q", r.Output)
	}
}

func TestGlob_SkipsGitDir(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "config.go"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte(""), 0o644)

	r := globFiles(map[string]string{
		"pattern":   "**/*.go",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if strings.Contains(r.Output, ".git") {
		t.Fatalf("should not include .git files, got: %s", r.Output)
	}
	lines := strings.Split(r.Output, "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 match, got %d: %s", len(lines), r.Output)
	}
}

func TestGlob_DefaultDirectory(t *testing.T) {
	r := globFiles(map[string]string{
		"pattern": "*.nonexistent_extension_xyz",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if r.Output != "No files matched" {
		t.Fatalf("expected 'No files matched', got %q", r.Output)
	}
}

// ── grep tests ──

func TestGrep_BasicMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main\n\nfunc hello() {}\nfunc world() {}\n"), 0o644)

	r := grepFiles(map[string]string{
		"pattern":   "func hello",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if !strings.Contains(r.Output, "func hello") {
		t.Fatalf("expected match for 'func hello', got: %s", r.Output)
	}
	if !strings.Contains(r.Output, ":3:") {
		t.Fatalf("expected line number 3, got: %s", r.Output)
	}
}

func TestGrep_RegexPattern(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("foo123\nbar456\nfoo789\n"), 0o644)

	r := grepFiles(map[string]string{
		"pattern":   "foo\\d+",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 matches, got %d: %s", len(lines), r.Output)
	}
}

func TestGrep_InvalidRegex(t *testing.T) {
	r := grepFiles(map[string]string{
		"pattern":   "[invalid",
		"directory": ".",
	}, 0)

	if r.Success {
		t.Fatal("expected failure for invalid regex")
	}
	if r.Error != "bad_pattern" {
		t.Fatalf("expected error 'bad_pattern', got %q", r.Error)
	}
}

func TestGrep_NoMatches(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("hello world\n"), 0o644)

	r := grepFiles(map[string]string{
		"pattern":   "nonexistent_string",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if r.Output != "No matches found" {
		t.Fatalf("expected 'No matches found', got %q", r.Output)
	}
}

func TestGrep_FileTypeFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.go"), []byte("match this\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "test.py"), []byte("match this\n"), 0o644)

	r := grepFiles(map[string]string{
		"pattern":   "match",
		"directory": dir,
		"file_type": "go",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if !strings.Contains(r.Output, "test.go") {
		t.Fatalf("expected test.go match, got: %s", r.Output)
	}
	if strings.Contains(r.Output, "test.py") {
		t.Fatalf("should not include test.py, got: %s", r.Output)
	}
}

func TestGrep_SkipsGitAndNodeModules(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	nmDir := filepath.Join(dir, "node_modules")
	os.MkdirAll(gitDir, 0o755)
	os.MkdirAll(nmDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("match this\n"), 0o644)
	os.WriteFile(filepath.Join(nmDir, "pkg.js"), []byte("match this\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("match this\n"), 0o644)

	r := grepFiles(map[string]string{
		"pattern":   "match",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if strings.Contains(r.Output, ".git") {
		t.Fatalf("should not include .git files, got: %s", r.Output)
	}
	if strings.Contains(r.Output, "node_modules") {
		t.Fatalf("should not include node_modules, got: %s", r.Output)
	}
	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 match, got %d: %s", len(lines), r.Output)
	}
}

func TestGrep_RecursiveSearch(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub", "deep")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(dir, "a.go"), []byte("findme\n"), 0o644)
	os.WriteFile(filepath.Join(sub, "b.go"), []byte("findme too\n"), 0o644)

	r := grepFiles(map[string]string{
		"pattern":   "findme",
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 matches, got %d: %s", len(lines), r.Output)
	}
}

// ── list_files tests ──

func TestListFiles_Basic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte(""), 0o644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)
	os.WriteFile(filepath.Join(dir, "subdir", "c.md"), []byte(""), 0o644)

	r := listFiles(map[string]string{
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	// Should have: a.txt, b.go, subdir/ (non-recursive, so not subdir/c.md)
	if len(lines) != 3 {
		t.Fatalf("expected 3 entries, got %d: %s", len(lines), r.Output)
	}

	// Check that paths are relative to the listed directory
	for _, line := range lines {
		if strings.Contains(line, "..") {
			t.Fatalf("paths should be relative to listed directory, got: %s", line)
		}
	}
}

func TestListFiles_WithExtensionFilter(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "b.go"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0o644)

	r := listFiles(map[string]string{
		"directory": dir,
		"extension": ".txt",
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 .txt files, got %d: %s", len(lines), r.Output)
	}
}

func TestListFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	r := listFiles(map[string]string{
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	if r.Output != "(empty directory)" {
		t.Fatalf("expected '(empty directory)', got %q", r.Output)
	}
}

func TestListFiles_SkipsHiddenAndGit(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte(""), 0o644)
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte(""), 0o644)

	r := listFiles(map[string]string{
		"directory": dir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 visible file, got %d: %s", len(lines), r.Output)
	}
	if !strings.Contains(r.Output, "a.txt") {
		t.Fatalf("expected a.txt, got: %s", r.Output)
	}
}

func TestListFiles_PathsRelativeToListedDirectory(t *testing.T) {
	// This is the key test for the bug fix
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0o755)
	os.WriteFile(filepath.Join(subdir, "file1.txt"), []byte(""), 0o644)
	os.WriteFile(filepath.Join(subdir, "file2.go"), []byte(""), 0o644)

	// Change to a different directory to test the bug
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	
	// Change to /tmp (or any directory not parent of dir)
	os.Chdir("/tmp")

	r := listFiles(map[string]string{
		"directory": subdir,
	}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}

	lines := strings.Split(strings.TrimSpace(r.Output), "\n")
	// Should have: file1.txt, file2.go (relative to subdir, not to cwd)
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.Contains(line, "..") || strings.Contains(line, "/") {
			t.Fatalf("paths should be simple filenames relative to listed directory, got: %s", line)
		}
		if line != "file1.txt" && line != "file2.go" {
			t.Fatalf("unexpected file: %s", line)
		}
	}
}

func TestListFiles_DefaultDirectory(t *testing.T) {
	// Test that it defaults to current directory when no directory specified
	r := listFiles(map[string]string{}, 0)

	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	// Should list files in current directory (which is the temp dir created by t.TempDir()
	// but the test runner changes to that dir, so it should work)
}
