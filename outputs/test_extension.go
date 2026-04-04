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
	testDir := "/tmp/test_listfiles_ext"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file3.py"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "README.md"), []byte("test"), 0644)
	
	// Test with .go extension filter
	result := tools.ExecuteDirect("list_files", map[string]string{
		"directory": testDir,
		"extension": ".go",
	}, 10)
	
	if !result.Success {
		fmt.Printf("Error: %s\n", result.Error)
		os.Exit(1)
	}
	
	fmt.Println("Output from list_files with extension=\".go\":")
	fmt.Println(result.Output)
	
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	if len(lines) == 1 && lines[0] == "file2.go" {
		fmt.Println("✓ Extension filter works correctly")
	} else {
		fmt.Printf("✗ Expected only 'file2.go', got: %v\n", lines)
		os.Exit(1)
	}
	
	// Test with .txt extension filter
	result2 := tools.ExecuteDirect("list_files", map[string]string{
		"directory": testDir,
		"extension": ".txt",
	}, 10)
	
	if !result2.Success {
		fmt.Printf("Error: %s\n", result2.Error)
		os.Exit(1)
	}
	
	fmt.Println("\nOutput from list_files with extension=\".txt\":")
	fmt.Println(result2.Output)
	
	lines2 := strings.Split(strings.TrimSpace(result2.Output), "\n")
	if len(lines2) == 1 && lines2[0] == "file1.txt" {
		fmt.Println("✓ Extension filter works correctly for .txt")
	} else {
		fmt.Printf("✗ Expected only 'file1.txt', got: %v\n", lines2)
		os.Exit(1)
	}
}