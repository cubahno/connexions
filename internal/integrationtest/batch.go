package integrationtest

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/cubahno/connexions/v2/pkg/config"
)

const (
	defaultBatchSizeBytes = 6 * 1024 * 1024 // 6MB
	defaultBatchMaxFiles  = 20
)

// SplitIntoBatches splits specs into batches based on target size in bytes.
// Each batch will contain specs until either target size or max files is reached.
// Large files (>= 50% of target) get their own batch for isolation.
func SplitIntoBatches(specs []string, targetBytes int64) [][]string {
	if targetBytes <= 0 {
		targetBytes = defaultBatchSizeBytes
	}

	largeFileThreshold := targetBytes / 2

	// Get file sizes
	type specWithSize struct {
		path string
		size int64
	}
	items := make([]specWithSize, 0, len(specs))
	for _, spec := range specs {
		size := int64(0)
		if info, err := os.Stat(spec); err == nil {
			size = info.Size()
		}
		items = append(items, specWithSize{spec, size})
	}

	// Sort by size descending - large files first for better packing
	sort.Slice(items, func(i, j int) bool {
		return items[i].size > items[j].size
	})

	// Separate large files (each gets own batch) from regular files
	var batches [][]string
	var batchSizes []int64
	var regularItems []specWithSize

	for _, item := range items {
		if item.size >= largeFileThreshold {
			// Large file gets its own batch
			batches = append(batches, []string{item.path})
			batchSizes = append(batchSizes, item.size)
		} else {
			regularItems = append(regularItems, item)
		}
	}

	// Greedy bin-packing for regular files (constrained by size AND file count)
	for _, item := range regularItems {
		placed := false
		for i := range batches {
			if batchSizes[i]+item.size <= targetBytes && len(batches[i]) < defaultBatchMaxFiles {
				batches[i] = append(batches[i], item.path)
				batchSizes[i] += item.size
				placed = true
				break
			}
		}
		if !placed {
			// Start new batch
			batches = append(batches, []string{item.path})
			batchSizes = append(batchSizes, item.size)
		}
	}

	return batches
}

// BatchServerInfo holds info about a built batch server
type BatchServerInfo struct {
	BatchID      int
	ServiceNames []string
	ServerBin    string
	ServerDir    string
}

const batchServerTemplate = `// Code generated for integration tests. DO NOT EDIT.
package main

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/cubahno/connexions/v2/pkg/api"
	"github.com/cubahno/connexions/v2/pkg/loader"

	// Imports to ensure dependencies are available
	_ "github.com/go-playground/validator/v10"
	_ "github.com/google/uuid"

	// Service imports
{{range .ServiceImports}}	_ "{{.}}"
{{end}})

func main() {
	logLevel := slog.LevelInfo
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// NewRouter includes all middleware (RequestID, RealIP, Logger, Duration, Recoverer, Timeout)
	router := api.NewRouter()

	_ = api.CreateHealthRoutes(router)
	_ = api.CreateHomeRoutes(router)
	_ = api.CreateServiceRoutes(router)

	loader.LoadAll(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = "2200"
	}
	addr := fmt.Sprintf(":%s", port)

	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting batch server on %s with %d services", addr, {{.ServiceCount}})
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Server failed to start: %v", err)
	}
}
`

// GenerateBatchServer generates a main.go that imports all services in the batch
func GenerateBatchServer(sandboxDir string, batchID int, serviceNames []string) (*BatchServerInfo, error) {
	// Get module name for imports
	moduleName := "github.com/cubahno/connexions/v2"

	// Build service imports using relative services path
	paths := config.NewPaths("")
	var serviceImports []string
	for _, name := range serviceNames {
		importPath := fmt.Sprintf("%s/%s/%s", moduleName, paths.Services, name)
		serviceImports = append(serviceImports, importPath)
	}

	// Generate main.go content
	tmpl, err := template.New("batch").Parse(batchServerTemplate)
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	data := struct {
		ServiceImports []string
		ServiceCount   int
	}{
		ServiceImports: serviceImports,
		ServiceCount:   len(serviceNames),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	// Create batch server directory
	batchDir := filepath.Join(sandboxDir, fmt.Sprintf(".batch-%d", batchID))
	if err := os.MkdirAll(batchDir, 0755); err != nil {
		return nil, fmt.Errorf("creating batch dir: %w", err)
	}

	// Write main.go
	mainPath := filepath.Join(batchDir, "main.go")
	if err := os.WriteFile(mainPath, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("writing main.go: %w", err)
	}

	return &BatchServerInfo{
		BatchID:      batchID,
		ServiceNames: serviceNames,
		ServerDir:    batchDir,
	}, nil
}

// BuildBatchServer builds the batch server binary
func BuildBatchServer(sandboxDir string, info *BatchServerInfo) error {
	serverBin := filepath.Join(sandboxDir, fmt.Sprintf("batch-server-%d", info.BatchID))

	cmd := exec.Command("go", "build", "-o", serverBin, ".")
	cmd.Dir = info.ServerDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w\nstderr: %s", err, stderr.String())
	}

	info.ServerBin = serverBin
	return nil
}
