package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func listFiles(params map[string]string) string {
	dir := params["directory"]
	if dir == "" {
		dir = params["path"]
	}
	if dir == "" {
		dir = params["dir"]
	}
	if dir == "" {
		dir = "."
	}
	ext := params["extension"]
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
		if len(files) > 200 {
			return fmt.Errorf("limit reached")
		}
		if ext != "" && filepath.Ext(name) != ext {
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
	return strings.Join(files, "\n")
}

func main() {
	// Create test directory structure
	testDir := "/tmp/test_listfiles_bug"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir+"/subdir", 0755)
	os.WriteFile(testDir+"/subdir/file1.txt", []byte("test"), 0644)
	os.WriteFile(testDir+"/subdir/file2.go", []byte("test"), 0644)
	
	// Change to a different directory to demonstrate the bug
	originalDir, _ := os.Getwd()
	os.Chdir("/tmp")
	defer os.Chdir(originalDir)
	
	// Test listing files in testDir/subdir
	result := listFiles(map[string]string{"directory": testDir + "/subdir"})
	fmt.Println("Current directory:", originalDir)
	fmt.Println("Listing directory:", testDir + "/subdir")
	fmt.Println("Result:")
	fmt.Println(result)
	
	// Expected: file1.txt, file2.go (relative to testDir/subdir)
	// Actual with bug: ../../tmp/test_listfiles_bug/subdir/file1.txt, etc.
}