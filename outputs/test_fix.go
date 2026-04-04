package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/AgentGuardHQ/shellforge/internal/tools"
)

func testListFiles() {
	// Create test directory structure
	testDir := "test_listfiles_fix"
	os.RemoveAll(testDir)
	os.MkdirAll(filepath.Join(testDir, "subdir", "deep"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file3.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "deep", "file4.txt"), []byte("test"), 0644)
	defer os.RemoveAll(testDir)
	
	// Test 1: List all files
	fmt.Println("Test 1: List all files in", testDir)
	result := tools.ExecuteDirect("list_files", map[string]string{
		"directory": testDir,
	}, 10)
	if !result.Success {
		fmt.Println("Error:", result.Error)
		return
	}
	fmt.Println("Output:")
	fmt.Println(result.Output)
	
	// Test 2: List with .txt extension filter
	fmt.Println("\nTest 2: List only .txt files in", testDir)
	result = tools.ExecuteDirect("list_files", map[string]string{
		"directory": testDir,
		"extension": ".txt",
	}, 10)
	if !result.Success {
		fmt.Println("Error:", result.Error)
		return
	}
	fmt.Println("Output:")
	fmt.Println(result.Output)
	
	// Test 3: List from a different directory (change cwd)
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	defer os.Chdir(originalDir)
	
	fmt.Println("\nTest 3: List files from different cwd")
	result = tools.ExecuteDirect("list_files", map[string]string{
		"directory": filepath.Join(originalDir, testDir),
	}, 10)
	if !result.Success {
		fmt.Println("Error:", result.Error)
		return
	}
	fmt.Println("Output:")
	fmt.Println(result.Output)
	
	// Check that paths are relative to the listed directory, not cwd
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	allRelative := true
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Paths should not start with testDir (they should be relative to it)
		if strings.HasPrefix(line, testDir) {
			fmt.Printf("ERROR: Path %s starts with %s (should be relative to directory)\n", line, testDir)
			allRelative = false
		}
	}
	if allRelative {
		fmt.Println("✓ All paths are correctly relative to the listed directory")
	}
}

func main() {
	testListFiles()
}