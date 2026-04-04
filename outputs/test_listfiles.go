package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func listFiles(dir string) []string {
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
		rel, _ := filepath.Rel(".", path)
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
	// Create test directory structure
	testDir := "/tmp/test_listfiles"
	os.RemoveAll(testDir)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file2.txt"), []byte("test"), 0644)
	
	// Get current directory
	cwd, _ := os.Getwd()
	fmt.Println("Current working directory:", cwd)
	
	// Call listFiles
	files := listFiles(testDir)
	fmt.Println("\nFiles returned (relative to cwd):")
	for _, f := range files {
		fmt.Println("  ", f)
	}
	
	// What we expect (relative to testDir):
	fmt.Println("\nExpected (relative to", testDir, "):")
	fmt.Println("  file1.txt")
	fmt.Println("  subdir/")
	fmt.Println("  subdir/file2.txt")
	
	// Show what filepath.Rel returns
	fmt.Println("\nDebug filepath.Rel('.', path):")
	filepath.WalkDir(testDir, func(path string, d os.DirEntry, err error) error {
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
		rel, _ := filepath.Rel(".", path)
		fmt.Printf("  %s -> %s\n", path, rel)
		return nil
	})
}