package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/AgentGuardHQ/shellforge/internal/tools"
)

func main() {
	// Create test directory structure
	testDir := "/tmp/test_glob_bug"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file1.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("test"), 0644)
	
	// Change to a different directory
	os.Chdir("/tmp")
	
	// Test glob with directory parameter
	result := tools.ExecuteDirect("glob", map[string]string{
		"pattern": "*.go",
		"directory": testDir,
	}, 10)
	
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
		os.Exit(1)
	}
	
	fmt.Println("Output from glob with directory parameter:")
	fmt.Println(result.Output)
	
	// The glob function might have a similar bug
	// Let me check what paths it returns
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	for _, line := range lines {
		if line == "No files matched" {
			continue
		}
		fmt.Printf("Path: %q\n", line)
		// Is it absolute or relative?
		if filepath.IsAbs(line) {
			fmt.Println("  - Absolute path")
		} else {
			fmt.Println("  - Relative path")
		}
	}
}