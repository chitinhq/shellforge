package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Create test directory structure
	testDir := "/tmp/test_listfiles_bug3"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir+"/subdir", 0755)
	os.WriteFile(testDir+"/subdir/file1.txt", []byte("test"), 0644)
	os.WriteFile(testDir+"/subdir/file2.go", []byte("test"), 0644)
	
	// Save original directory
	originalDir, _ := os.Getwd()
	fmt.Println("Original directory:", originalDir)
	
	// Change to a completely different directory
	os.Chdir("/usr")
	currentDir, _ := os.Getwd()
	fmt.Println("Current directory after chdir:", currentDir)
	
	// Simulate the bug in listFiles
	dir := testDir + "/subdir"
	fmt.Println("\nWalking directory:", dir)
	
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			fmt.Printf("Error at %s: %v\n", path, err)
			return nil
		}
		fmt.Printf("Visiting: %s, IsDir: %v\n", path, d.IsDir())
		if d.IsDir() {
			return nil
		}
		// This is the bug: using "." instead of dir
		rel, err := filepath.Rel(".", path)
		if err != nil {
			fmt.Printf("Error computing rel for %s: %v\n", path, err)
		} else {
			fmt.Printf("  Relative to '.': %s\n", rel)
		}
		return nil
	})
}