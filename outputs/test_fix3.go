package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func testListFiles(dir string) {
	fmt.Printf("\n=== Testing with dir=%q ===\n", dir)
	
	var files []string
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		fmt.Printf("  Visiting: path=%q, name=%q, isDir=%v\n", path, name, d.IsDir())
		
		if name == "node_modules" || name == ".git" || strings.HasPrefix(name, ".") {
			fmt.Printf("    Skipping because name starts with dot or is .git/node_modules\n")
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		fmt.Printf("    rel=%q\n", rel)
		if rel == "." {
			// Skip the root directory itself
			fmt.Printf("    Skipping root\n")
			return nil
		}
		if d.IsDir() {
			files = append(files, rel+"/")
		} else {
			files = append(files, rel)
		}
		return nil
	})
	
	fmt.Println("Files:")
	for _, f := range files {
		fmt.Printf("  %q\n", f)
	}
}

func main() {
	// Create test directory structure
	testDir := "/tmp/test_listfiles_fix3"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file3.py"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file4.md"), []byte("test"), 0644)
	
	// Test with "."
	os.Chdir(testDir)
	testListFiles(".")
}