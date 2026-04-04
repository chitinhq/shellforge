package main

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/AgentGuardHQ/shellforge/internal/tools"
)

func main() {
	// Create test directory structure
	tmpDir := "/tmp/test_fix"
	os.RemoveAll(tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "sub"), 0755)
	
	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "a.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "sub", "c.txt"), []byte("test"), 0644)
	
	// Change to different directory
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	defer os.Chdir(originalDir)
	
	// Test the fixed listFiles
	result := tools.ExecuteDirect("list_files", map[string]string{
		"directory": tmpDir,
	}, 0)
	
	fmt.Println("Testing fixed listFiles function:")
	fmt.Println("Directory:", tmpDir)
	fmt.Println("Current working directory:", "/tmp")
	fmt.Println("\nResult:")
	if result.Success {
		fmt.Println(result.Output)
	} else {
		fmt.Printf("Error: %s\n", result.Error)
	}
	
	// Also test with extension filter
	fmt.Println("\n--- Testing with .txt extension filter ---")
	result2 := tools.ExecuteDirect("list_files", map[string]string{
		"directory": tmpDir,
		"extension": ".txt",
	}, 0)
	
	if result2.Success {
		fmt.Println(result2.Output)
	} else {
		fmt.Printf("Error: %s\n", result2.Error)
	}
	
	// Clean up
	os.RemoveAll(tmpDir)
}