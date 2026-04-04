package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFiles_RelativeToDirectory(t *testing.T) {
	// Create a test directory structure
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	os.MkdirAll(subDir, 0755)
	
	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(subDir, "file3.txt"), []byte("test"), 0644)
	
	// Change to a different directory to test that paths are relative to dir, not cwd
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	defer os.Chdir(originalDir)
	
	// Test listing the directory
	r := listFiles(map[string]string{
		"directory": tmpDir,
	}, 0)
	
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	
	output := r.Output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Check that we get relative paths (not absolute paths)
	for _, line := range lines {
		if strings.HasPrefix(line, "/") {
			t.Errorf("path should be relative to directory, not absolute: %s", line)
		}
		if strings.Contains(line, "tmp") && filepath.IsAbs(line) {
			t.Errorf("path should be relative, not contain absolute path segments: %s", line)
		}
	}
	
	// Check expected files
	expectedFiles := []string{"./", "file1.txt", "file2.go", "subdir/", "subdir/file3.txt"}
	foundCount := 0
	for _, expected := range expectedFiles {
		for _, line := range lines {
			if line == expected {
				foundCount++
				break
			}
		}
	}
	
	if foundCount != len(expectedFiles) {
		t.Errorf("expected to find %d files, found %d. Output:\n%s", len(expectedFiles), foundCount, output)
	}
}

func TestListFiles_WithExtensionFilter(t *testing.T) {
	tmpDir := t.TempDir()
	
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file3.txt"), []byte("test"), 0644)
	
	r := listFiles(map[string]string{
		"directory": tmpDir,
		"extension": ".txt",
	}, 0)
	
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	
	output := r.Output
	lines := strings.Split(strings.TrimSpace(output), "\n")
	
	// Should only get .txt files
	for _, line := range lines {
		if line == "./" {
			continue // directory entry
		}
		if !strings.HasSuffix(line, ".txt") {
			t.Errorf("expected only .txt files, got: %s", line)
		}
	}
	
	// Should have 2 .txt files
	txtCount := 0
	for _, line := range lines {
		if strings.HasSuffix(line, ".txt") {
			txtCount++
		}
	}
	
	if txtCount != 2 {
		t.Errorf("expected 2 .txt files, got %d. Output:\n%s", txtCount, output)
	}
}

func TestListFiles_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	
	r := listFiles(map[string]string{
		"directory": tmpDir,
	}, 0)
	
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	
	if r.Output != "(empty directory)" {
		t.Errorf("expected '(empty directory)', got: %s", r.Output)
	}
}

func TestListFiles_DefaultDirectory(t *testing.T) {
	// Test with empty directory parameter (should default to ".")
	r := listFiles(map[string]string{}, 0)
	
	if !r.Success {
		t.Fatalf("expected success, got error: %s", r.Error)
	}
	
	// Should at least not crash
	if strings.Contains(r.Error, "list_error") {
		t.Errorf("should not have list_error, got: %s", r.Error)
	}
}