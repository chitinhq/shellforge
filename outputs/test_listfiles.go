package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Create a test directory structure
	tmpDir := "/tmp/test_listfiles"
	os.RemoveAll(tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	
	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("test"), 0644)
	
	// Test current behavior
	fmt.Println("Current directory:", tmpDir)
	
	// Change to a different directory to test
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	defer os.Chdir(originalDir)
	
	// Simulate what listFiles does
	dir := tmpDir
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
		// Current implementation
		relCurrent, _ := filepath.Rel(".", path)
		// Proposed fix
		relFixed, _ := filepath.Rel(dir, path)
		
		if d.IsDir() {
			files = append(files, fmt.Sprintf("Current: %s/, Fixed: %s/", relCurrent, relFixed))
		} else {
			files = append(files, fmt.Sprintf("Current: %s, Fixed: %s", relCurrent, relFixed))
		}
		return nil
	})
	
	fmt.Println("Results:")
	for _, f := range files {
		fmt.Println(f)
	}
}