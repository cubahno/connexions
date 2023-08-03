package integrationtest

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

const (
	// SandboxDir is the directory where all test artifacts are created
	SandboxDir = ".sandbox"
)

// CleanupSandbox removes the sandbox directory.
func CleanupSandbox() error {
	if err := os.RemoveAll(SandboxDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to cleanup sandbox: %w", err)
	}
	log.Printf("Cleaned up sandbox: %s\n", SandboxDir)
	return nil
}

// CreateSandbox creates the sandbox directory.
// Returns an absolute path to the sandbox directory.
func CreateSandbox() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	sandboxDir := filepath.Join(cwd, SandboxDir)

	if err := os.MkdirAll(sandboxDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create sandbox: %w", err)
	}

	return sandboxDir, nil
}
