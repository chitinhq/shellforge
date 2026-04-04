package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Simulating the fixed listFiles function
func listFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		// Skip node_modules, .git, and hidden files/directories (starting with .)
		// but exclude "." and ".." which are special directory entries
		if name == "node_modules" || name == ".git" || (strings.HasPrefix(name, ".") && name != "." && name != "..") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if len(files) > 200 {
			return fmt.Errorf("limit reached")
		}
		rel, _ := filepath.Rel(dir, path)
		// Skip the root directory itself (rel == ".")
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
	return files, err
}

func main() {
	// Create test directory structure
	testDir := "/tmp/test_listfiles_final"
	os.RemoveAll(testDir)
	os.MkdirAll(testDir, 0755)
	os.MkdirAll(filepath.Join(testDir, "subdir"), 0755)
	os.MkdirAll(filepath.Join(testDir, ".hidden_dir"), 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "file2.go"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, ".hidden_file"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file3.py"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(testDir, "subdir", "file4.md"), []byte("test"), 0644)
	
	fmt.Println("=== Test 1: Absolute path ===")
	files, err := listFiles(testDir)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for _, f := range files {
			fmt.Printf("  %q\n", f)
		}
	}
	
	fmt.Println("\n=== Test 2: Relative path from /tmp ===")
	os.Chdir("/tmp")
	files, err = listFiles("test_listfiles_final")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for _, f := range files {
			fmt.Printf("  %q\n", f)
		}
	}
	
	fmt.Println("\n=== Test 3: Current directory (.) ===")
	os.Chdir(testDir)
	files, err = listFiles(".")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for _, f := range files {
			fmt.Printf("  %q\n", f)
		}
	}
	
	fmt.Println("\n=== Test 4: Parent directory ===")
	os.Chdir(filepath.Join(testDir, "subdir"))
	files, err = listFiles("..")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		for _, f := range files {
			fmt.Printf("  %q\n", f)
		}
	}
}