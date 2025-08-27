package testhelpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
)

var (
	_, b, _, _   = runtime.Caller(0)
	testDataPath = filepath.Join(filepath.Dir(b), "..", "..", "testdata")
)

var (
	pluginOnce sync.Once
	PluginPath string
)

func CreateTestPlugin() string {
	var err error
	pluginOnce.Do(func() {
		PluginPath, err = createPlugin()
		if err != nil {
			panic(err)
		}
	})
	return PluginPath
}

func createPlugin() (string, error) {
	dir := os.TempDir()
	codeDir := filepath.Join(testDataPath, "plugins")

	soName := "userlib.so"
	pluginPath := filepath.Join(dir, soName)

	// Build the user code into a shared library
	// Change working directory to the temporary directory
	cmdArgs := []string{"build", "-buildmode=plugin"}

	// Check if the environment variable is set
	if os.Getenv("DEBUG_BUILD") == "true" {
		cmdArgs = append(cmdArgs, "-gcflags", "all=-N -l")
	}

	cmdArgs = append(cmdArgs, "-o", pluginPath, codeDir)

	cmd := exec.Command("go", cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GOARCH="+runtime.GOARCH,
		"GOOS="+runtime.GOOS,
		"CGO_ENABLED=1",
		"GO111MODULE=on",
	)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return pluginPath, nil
}
