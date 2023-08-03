package files

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetModuleInfo walks up the directory tree to find go.mod and returns both
// the module name and the module root directory path
func GetModuleInfo(startDir string) (string, string, error) {
	dir := startDir
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Found go.mod, read the module name and return the directory
			moduleName, err := getModuleName(goModPath)
			if err != nil {
				return "", "", err
			}
			return moduleName, dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached the root directory
			return "", "", fmt.Errorf("go.mod not found in %s or any parent directory", startDir)
		}
		dir = parent
	}
}

// getModuleName reads the module name from a go.mod file
func getModuleName(goModPath string) (string, error) {
	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %w", err)
	}

	return "", fmt.Errorf("module declaration not found in go.mod")
}
