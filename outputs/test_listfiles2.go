package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func listFilesCurrent(dir string) []string {
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

func listFilesFixed(dir string) []string {
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
		// Fixed implementation - relative to dir parameter
		rel, _ := filepath.Rel(dir, path)
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
	// Create test directory structure in current directory
	testDir := "test_listfiles_dir"
	os.RemoveAll(testDir)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file2.txt"), []byte("test"), 0644)
	defer os.RemoveAll(testDir)
	
	// Get current directory
	cwd, _ := os.Getwd()
	fmt.Println("Current working directory:", cwd)
	fmt.Println("Test directory:", testDir)
	
	// Call current implementation
	filesCurrent := listFilesCurrent(testDir)
	fmt.Println("\nCurrent implementation (relative to cwd):")
	for _, f := range filesCurrent {
		fmt.Println("  ", f)
	}
	
	// Call fixed implementation
	filesFixed := listFilesFixed(testDir)
	fmt.Println("\nFixed implementation (relative to", testDir, "):")
	for _, f := range filesFixed {
		fmt.Println("  ", f)
	}
	
	// What we expect
	fmt.Println("\nExpected (relative to", testDir, "):")
	fmt.Println("  . (or empty for root dir)")
	fmt.Println("  file1.txt")
	fmt.Println("  subdir/")
	fmt.Println("  subdir/file2.txt")
}