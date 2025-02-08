package main

import (
	"fmt"
	"os"
	"os/exec"
	"plugin"
	"regexp"
	"strings"
)

// Injected at build time
var goVersion string

func main() {
	pluginPath := ""
	if len(os.Args) > 1 {
		pluginPath = os.Args[1]
	}

	if pluginPath == "" {
		fmt.Println("❌ ERROR: No plugin path provided. Exiting...")
		os.Exit(1)
	}

	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		fmt.Printf("⚠️ WARNING: Plugin not found at %s. Exiting...\n", pluginPath)
		os.Exit(1)
	}

	// Try loading the plugin
	_, err := plugin.Open(pluginPath)
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to load plugin %s: %v\n", pluginPath, err)
		fmt.Printf("📦 VERSION: Make sure your go version is: %s\n", goVersion)
		fmt.Println("🚩FLAGS: ...")
		os.Exit(1)
	}

	// Run `go tool nm` to get symbols
	cmd := exec.Command("go", "tool", "nm", pluginPath)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ ERROR: Failed to extract symbols: %v\n", err)
		os.Exit(1)
	}

	// Extract function names in the `plugin/unnamed-<hash>` namespace
	fmt.Println("✅ Plugin should run without problems within connexions.")
	fmt.Println("🔍 Functions:")
	pluginFuncPattern := regexp.MustCompile(` T plugin/[^.]+\.(.+)$`)
	for _, line := range strings.Split(string(output), "\n") {
		if matches := pluginFuncPattern.FindStringSubmatch(line); matches != nil {
			fmt.Printf("   - %s\n", matches[1]) // Print only function name
		}
	}
}
