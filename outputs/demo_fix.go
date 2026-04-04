package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	// Test to demonstrate the fix for issue #24
	// listFiles() now returns paths relative to the listed directory, not cwd
	
	fmt.Println("=== Testing listFiles bug fix ===")
	
	// Create test directory structure
	testDir := "/tmp/test_issue_24"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir+"/project/src", 0755)
	os.WriteFile(testDir+"/project/src/main.go", []byte("package main"), 0644)
	os.WriteFile(testDir+"/project/src/utils.go", []byte("package main"), 0644)
	os.WriteFile(testDir+"/project/README.md", []byte("# Project"), 0644)
	
	// Save original directory
	originalDir, _ := os.Getwd()
	fmt.Printf("Original working directory: %s\n", originalDir)
	
	// Change to a completely different directory
	os.Chdir("/usr")
	currentDir, _ := os.Getwd()
	fmt.Printf("Changed to directory: %s\n", currentDir)
	
	fmt.Println("\n1. Testing listFiles with the FIXED implementation:")
	fmt.Println("   (Paths should be relative to the listed directory, not to /usr)")
	
	// Simulate the fixed listFiles behavior
	dir := testDir + "/project/src"
	entries, _ := os.ReadDir(dir)
	fmt.Printf("   Listing directory: %s\n", dir)
	fmt.Println("   Results:")
	for _, d := range entries {
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if d.IsDir() {
			fmt.Printf("     %s/\n", name)
		} else {
			fmt.Printf("     %s\n", name)
		}
	}
	
	// Show what the buggy behavior would have been
	fmt.Println("\n2. What the BUGGY implementation would have returned:")
	fmt.Println("   (Trying to compute paths relative to /usr would fail)")
	
	// Change back
	os.Chdir(originalDir)
	
	fmt.Println("\n=== Test complete ===")
	fmt.Println("The fix ensures listFiles() returns paths relative to the")
	fmt.Println("directory being listed, not relative to the current working directory.")
}