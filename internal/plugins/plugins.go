package plugins

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"runtime"
	"time"
)

// CompilePlugin compiles user-provided Go code into a shared library.
// In debugging mode, add DEBUG_BUILD=true to the environment variables.
func CompilePlugin(dir string) (*plugin.Plugin, error) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "usergo")
	if err != nil {
		return nil, err
	}
	// Clean up
	defer os.RemoveAll(tmpDir)
	soName := "userlib.so"
	pluginPath := filepath.Join(tmpDir, soName)
	// unique module name needed to avoid `plugin already loaded` error
	moduleName := fmt.Sprintf("middleware_%d", time.Now().UnixNano())

	// Copy user-provided Go files into the temporary directory
	numCopied := 0
	if err := filepath.Walk(dir, func(src string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		dst := filepath.Join(tmpDir, info.Name())
		content, err := os.ReadFile(src)
		if err != nil {
			return err
		}

		if err = os.WriteFile(dst, content, 0644); err != nil {
			return err
		}
		numCopied += 1
		return nil
	}); err != nil {
		return nil, err
	}

	if numCopied == 0 {
		return nil, nil
	}

	if err := initModuleIfNone(tmpDir, moduleName); err != nil {
		return nil, fmt.Errorf("failed to initialize module: %v", err)
	}

	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmpDir

	// Create buffers to capture output and errors
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to tidy callback modules: %v", err)
	}

	// Build the user code into a shared library
	// Change working directory to the temporary directory
	cmdArgs := []string{"build", "-buildmode=plugin"}

	// Check if the environment variable is set
	if os.Getenv("DEBUG_BUILD") == "true" {
		cmdArgs = append(cmdArgs, "-gcflags", "all=-N -l")
	}

	cmdArgs = append(cmdArgs, "-o", soName, ".")

	cmd = exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GOROOT="+runtime.GOROOT(),
		"GOARCH="+runtime.GOARCH,
		"GOOS="+runtime.GOOS,
		"CGO_ENABLED=1",
		"GO111MODULE=on",
	)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("GOROOT:", runtime.GOROOT())
	fmt.Println("GOARCH:", runtime.GOARCH)
	fmt.Println("GOOS:", runtime.GOOS)

	cmd.Dir = tmpDir
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to build middleware: %s", out.String())
	}

	log.Printf("Loading plugin from: %s\n", pluginPath)
	return plugin.Open(pluginPath)
}

func initModuleIfNone(tmpDir, name string) error {
	goModPath := tmpDir + "/go.mod"

	if _, err := os.Stat(goModPath); err == nil {
		// go.mod already exists, nothing to do
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check for go.mod: %v", err)
	}

	// Initialize module with a name
	cmd := exec.Command("go", "mod", "init", name)
	cmd.Dir = tmpDir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize module: %s", out.String())
	}

	return nil
}
