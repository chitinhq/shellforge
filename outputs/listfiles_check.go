package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Simulate the fixed listFiles function
func listFiles(dir string) string {
	var files []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		// Skip the root directory itself
		if rel == "." {
			return nil
		}
		if d.IsDir() {
			files = append(files, rel+"/")
		} else {
			files = append(files, rel)
		}
		return nil
	})
	if len(files) == 0 {
		return "(empty directory)"
	}
	return strings.Join(files, "\n")
}

func TestListFiles_Basic(t *testing.T) {
	dir := t.TempDir()
	
	// Create test files
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.go"), []byte("test"), 0644)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "subdir", "file3.txt"), []byte("test"), 0644)
	
	// Change to a different directory to test that paths are relative to dir, not cwd
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	defer os.Chdir(originalDir)
	
	output := listFiles(dir)
	
	lines := strings.Split(strings.TrimSpace(output), "\n")
	expectedFiles := map[string]bool{
		"file1.txt": true,
		"file2.go": true,
		"subdir/": true,
		"subdir/file3.txt": true,
	}
	
	for _, line := range lines {
		if !expectedFiles[line] {
			t.Errorf("unexpected file in output: %q", line)
		}
		delete(expectedFiles, line)
	}
	
	for file := range expectedFiles {
		t.Errorf("missing expected file: %q", file)
	}
}

func main() {
	// Run a quick manual test
	dir := "/tmp/test_manual"
	os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(dir, "subdir", "file3.txt"), []byte("test"), 0644)
	
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	output := listFiles(dir)
	os.Chdir(originalDir)
	
	fmt.Println("Manual test output:")
	fmt.Println(output)
}