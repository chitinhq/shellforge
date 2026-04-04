package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Test the current implementation
	dir := "/tmp/test_listfiles"
	
	// Get current directory
	cwd, _ := os.Getwd()
	fmt.Printf("Current directory: %s\n", cwd)
	fmt.Printf("Target directory: %s\n\n", dir)
	
	// Simulate what listFiles does
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
		rel, _ := filepath.Rel(".", path)
		if d.IsDir() {
			files = append(files, rel+"/")
		} else {
			files = append(files, rel)
		}
		return nil
	})
	
	fmt.Println("Current implementation (relative to .):")
	for _, f := range files {
		fmt.Printf("  %q\n", f)
	}
	
	// What it should do
	files = []string{}
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
		if rel == "." {
			// Skip the root directory itself
			return nil
		}
		if d.IsDir() {
			files = append(files, rel+"/")
		} else {
			files = append(files, rel)
		}
		return nil
	})
	
	fmt.Println("\nShould be (relative to dir):")
	for _, f := range files {
		fmt.Printf("  %q\n", f)
	}
}