package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Simulated fixed listFiles function
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
		// FIXED: Use dir instead of "."
		rel, _ := filepath.Rel(dir, path)
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
	testDir := "/tmp/test_listfiles_fixed"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir+"/subdir", 0755)
	os.WriteFile(testDir+"/subdir/file1.txt", []byte("test"), 0644)
	os.WriteFile(testDir+"/subdir/file2.go", []byte("test"), 0644)
	os.WriteFile(testDir+"/subdir/.hidden", []byte("test"), 0644)
	os.MkdirAll(testDir+"/subdir/.git", 0755)
	os.WriteFile(testDir+"/subdir/.git/config", []byte("test"), 0644)
	
	// Save original directory
	originalDir, _ := os.Getwd()
	fmt.Println("Original directory:", originalDir)
	
	// Change to a completely different directory
	os.Chdir("/usr")
	currentDir, _ := os.Getwd()
	fmt.Println("Current directory after chdir:", currentDir)
	
	// Test listing files
	result := listFiles(map[string]string{"directory": testDir + "/subdir"})
	fmt.Println("\nListing directory:", testDir+"/subdir")
	fmt.Println("Result:")
	fmt.Println(result)
	
	// Test with extension filter
	result2 := listFiles(map[string]string{"directory": testDir + "/subdir", "extension": ".txt"})
	fmt.Println("\nWith extension filter (.txt):")
	fmt.Println(result2)
	
	// Test with extension filter for .go files
	result3 := listFiles(map[string]string{"directory": testDir + "/subdir", "extension": ".go"})
	fmt.Println("\nWith extension filter (.go):")
	fmt.Println(result3)
	
	// Change back
	os.Chdir(originalDir)
}