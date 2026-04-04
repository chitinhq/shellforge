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
	testDir := "/tmp/test_listfiles_bug"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file3.py"), []byte("test"), 0644)
	
	// Change to a different directory
	os.Chdir("/tmp")
	
	// Test listFiles with the directory parameter
	result := tools.ExecuteDirect("list_files", map[string]string{
		"directory": testDir,
	}, 10)
	
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
		os.Exit(1)
	}
	
	fmt.Println("Output from list_files:")
	fmt.Println(result.Output)
	
	// Check that paths are relative to testDir, not to /tmp
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	allCorrect := true
	for _, line := range lines {
		if line == "(empty directory)" {
			continue
		}
		// Paths should NOT start with /tmp/test_listfiles_bug/
		// They should be relative paths like "file1.txt" or "subdir/file3.py"
		if strings.HasPrefix(line, "/") {
			fmt.Printf("ERROR: Path %q is absolute, should be relative to %s\n", line, testDir)
			allCorrect = false
		}
		if strings.Contains(line, "/tmp/") {
			fmt.Printf("ERROR: Path %q contains /tmp/, should be relative to %s\n", line, testDir)
			allCorrect = false
		}
	}
	
	if allCorrect {
		fmt.Println("\n✓ All paths are correctly relative to the specified directory")
	} else {
		fmt.Println("\n✗ Some paths are incorrect")
		os.Exit(1)
	}
	
	// Also test with "."
	os.Chdir(testDir)
	result2 := tools.ExecuteDirect("list_files", map[string]string{
		"directory": ".",
	}, 10)
	
	if !result2.Success {
		fmt.Printf("Error with .: %s\n", result2.Error)
		os.Exit(1)
	}
	
	fmt.Println("\nOutput from list_files with directory=\".\":")
	fmt.Println(result2.Output)
	
	lines2 := strings.Split(strings.TrimSpace(result2.Output), "\n")
	if len(lines2) > 0 && lines2[0] != "(empty directory)" {
		fmt.Println("\n✓ list_files with directory=\".\" works correctly")
	} else {
		fmt.Println("\n✗ list_files with directory=\".\" returned empty")
		os.Exit(1)
	}
}