package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func simulateListFilesFixedSkipDot(dir string) []string {
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
	return files
}

func main() {
	// Create a test directory structure
	tmpDir := "/tmp/test_listfiles_skipdot"
	os.RemoveAll(tmpDir)
	os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755)
	
	// Create some test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("test"), 0644)
	
	fmt.Println("Test directory:", tmpDir)
	
	// Change to a different directory to test
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	
	files := simulateListFilesFixedSkipDot(tmpDir)
	fmt.Println("\nFiles returned (fixed behavior, skipping .):")
	for _, f := range files {
		fmt.Println("  ", f)
	}
	
	os.Chdir(originalDir)
}