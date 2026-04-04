package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Create test directory structure
	testDir := "/tmp/test_listfiles_bug"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir+"/subdir", 0755)
	os.WriteFile(testDir+"/subdir/file1.txt", []byte("test"), 0644)
	os.WriteFile(testDir+"/subdir/file2.go", []byte("test"), 0644)
	
	// Save original directory
	originalDir, _ := os.Getwd()
	fmt.Println("Original directory:", originalDir)
	
	// Change to a different directory to demonstrate the bug
	os.Chdir("/tmp")
	currentDir, _ := os.Getwd()
	fmt.Println("Current directory after chdir:", currentDir)
	
	// Simulate the bug in listFiles
	dir := testDir + "/subdir"
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
		if d.IsDir() {
			return nil
		}
		// This is the bug: using "." instead of dir
		rel, _ := filepath.Rel(".", path)
		files = append(files, rel)
		return nil
	})
	
	fmt.Println("Listing directory:", dir)
	fmt.Println("Result (with bug):")
	for _, f := range files {
		fmt.Println("  ", f)
	}
	
	// What it should be:
	var correctFiles []string
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
		if d.IsDir() {
			return nil
		}
		// Correct: use dir instead of "."
		rel, _ := filepath.Rel(dir, path)
		correctFiles = append(correctFiles, rel)
		return nil
	})
	
	fmt.Println("\nExpected result (relative to listed directory):")
	for _, f := range correctFiles {
		fmt.Println("  ", f)
	}
}