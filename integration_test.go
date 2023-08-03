package connexions

import (
	"embed"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/cubahno/connexions/v2/internal/integrationtest"
	"github.com/cubahno/connexions/v2/pkg/api"
)

//go:embed testdata/specs
var specsFS embed.FS

const (
	specsBaseDir = "testdata/specs"
	basePort     = 19000 // Base port for per-service servers
)

var (
	maxFails = 500
)

// TestIntegration tests each service independently with its own server instance.
// Each service runs the full pipeline: setup → generate → build → start → test → stop
// Each service gets its own server binary and runs on a unique port.
//
// Environment Variables:
//   - SPEC: Path to a specific spec file to test
//   - SPECS: Space-separated list of spec files to test
//   - MAX_CONCURRENCY: Maximum parallel service tests (default: 4)
//   - MAX_FAILS: Maximum failures before aborting (default: 200)
//   - CODEGEN_CONFIG: Path to custom codegen config
//   - SERVICE_CONFIG: Path to custom service config
//   - BATCH_SIZE: Number of services per batch (default: 50, 0 = disable batching)
//
// Run with: go test -v -tags=integration -run TestIntegration
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	integrationtest.SetSpecsFS(specsFS)

	// Load runtime options from environment
	runtimeOpts := integrationtest.NewRuntimeOptionsFromEnv()
	runtimeOpts.SpecsBaseDir = specsBaseDir

	// Collect specs
	var specPaths []string
	if spec := os.Getenv("SPEC"); spec != "" {
		specPaths = append(specPaths, spec)
	}
	if specs := os.Getenv("SPECS"); specs != "" {
		specPaths = append(specPaths, strings.Fields(specs)...)
	}

	specs := integrationtest.CollectSpecs(t, specPaths)
	if len(specs) == 0 {
		fmt.Fprintf(os.Stderr, "No specs to process\n")
		return
	}

	// Filter out specs larger than max size
	specs, excluded := integrationtest.FilterSpecsBySize(specs, runtimeOpts.MaxSpecSizeBytes)
	if excluded > 0 {
		fmt.Fprintf(os.Stderr, "Excluded %d specs larger than %dMB\n", excluded, runtimeOpts.MaxSpecSizeMB)
	}

	// Random sampling
	if runtimeOpts.RandomSpecs > 0 && runtimeOpts.RandomSpecs < len(specs) {
		rand.Shuffle(len(specs), func(i, j int) {
			specs[i], specs[j] = specs[j], specs[i]
		})
		specs = specs[:runtimeOpts.RandomSpecs]
		fmt.Fprintf(os.Stderr, "Randomly selected %d specs\n", runtimeOpts.RandomSpecs)
	}

	fmt.Fprintf(os.Stderr, "Found %d spec(s) to process\n", len(specs))

	// Create sandbox
	log.Printf("🔧 Setting up sandbox...\n")
	setupStart := time.Now()

	if integrationtest.ShouldCleanSandbox() {
		log.Println("🧹 Cleaning sandbox (uncommitted changes or forced)")
		if err := integrationtest.CleanupSandbox(); err != nil {
			t.Fatalf("Failed to cleanup sandbox: %v", err)
		}
	} else {
		log.Println("♻️  Reusing existing sandbox")
	}

	sandboxDir, err := integrationtest.CreateSandbox()
	if err != nil {
		t.Fatalf("Failed to create sandbox: %v", err)
	}

	if err := integrationtest.SetupSandbox(sandboxDir); err != nil {
		t.Fatalf("Failed to setup sandbox: %v", err)
	}

	log.Printf("🔧 Sandbox created at: %s (%.1fs)\n", sandboxDir, time.Since(setupStart).Seconds())

	maxFails = runtimeOpts.MaxFails

	// Use batched pipeline for multiple specs, individual mode for single spec or when disabled
	if runtimeOpts.BatchSizeBytes > 0 && len(specs) > 1 {
		runBatchedPipeline(t, specs, sandboxDir, runtimeOpts)
	} else {
		runIndividualPipeline(t, specs, sandboxDir, runtimeOpts)
	}
}

// runBatchedPipeline runs tests using batched builds with pipelining
func runBatchedPipeline(t *testing.T, specs []string, sandboxDir string, runtimeOpts *integrationtest.RuntimeOptions) {
	// Load or create cache (in project root, not sandbox which gets cleaned)
	var cache *integrationtest.ResultCache
	if !runtimeOpts.NoCache {
		var err error
		cache, err = integrationtest.NewResultCache(".")
		if err != nil {
			log.Printf("⚠️  Failed to load cache: %v\n", err)
		} else if runtimeOpts.ClearCache {
			if err := cache.Clear(); err != nil {
				log.Printf("⚠️  Failed to clear cache: %v\n", err)
			} else {
				log.Printf("🗑️  Cache cleared\n")
			}
		} else if cache.Size() > 0 {
			originalCount := len(specs)
			specs = cache.FilterUncached(specs)
			skipped := originalCount - len(specs)
			if skipped > 0 {
				log.Printf("📦 Skipping %d cached passing specs (%d remaining)\n", skipped, len(specs))
			}
		}
	}

	if len(specs) == 0 {
		log.Printf("✅ All specs cached as passing. Use CLEAR_CACHE=1 to retest.\n")
		return
	}

	batches := integrationtest.SplitIntoBatches(specs, runtimeOpts.BatchSizeBytes)
	log.Printf("Testing %d services in %d batches (target=%dMB, max %d failures before abort)...\n",
		len(specs), len(batches), runtimeOpts.BatchSizeMB, maxFails)

	// Set up interrupt handling - ignore default behavior so we can handle gracefully
	interruptCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Ignore(os.Interrupt, syscall.SIGTERM) // Prevent go test from killing us
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("\n⚠️  Interrupted! Printing partial results...\n")
		close(interruptCh)
	}()
	defer signal.Reset(os.Interrupt, syscall.SIGTERM) // Restore default on exit

	cfg := &integrationtest.PipelineConfig{
		SandboxDir:   sandboxDir,
		RuntimeOpts:  runtimeOpts,
		BasePort:     basePort,
		MaxFails:     maxFails,
		TotalSpecs:   len(specs),
		TotalBatches: len(batches),
		InterruptCh:  interruptCh,
		Cache:        cache,
		OnResult: func(result integrationtest.IntegrationResult, stats *integrationtest.ServiceStats, completed, total int) {
			status := "✅"
			if stats.Fails > 0 {
				status = "❌"
			}
			log.Printf("  [%d/%d] %s %s (%d/%d ok, %.1fs)\n",
				completed, total, status, stats.Name, stats.Success, stats.Tested, stats.TTE.Seconds())
		},
		OnBatchStart: func(batchID, totalBatches, servicesInBatch int, batchSizeBytes int64) {
			sizeMB := float64(batchSizeBytes) / (1024 * 1024)
			log.Printf("📦 Batch %d/%d (%d services, %.1f MB)\n", batchID, totalBatches, servicesInBatch, sizeMB)
		},
		OnBatchPhase: func(batchID int, phase string, done bool, elapsed time.Duration) {
			if done {
				log.Printf("  📦 Batch %d: %s done (%.1fs)\n", batchID, phase, elapsed.Seconds())
			}
		},
		OnBatchDone: func(batchID, totalBatches, servicesInBatch int, buildTime, testTime time.Duration) {
			log.Printf("📦 Batch %d/%d done (build: %.1fs, test: %.1fs)\n",
				batchID, totalBatches, buildTime.Seconds(), testTime.Seconds())
		},
	}

	allResults, serviceStatsMap, totalBuildTime, batchStats := integrationtest.RunPipeline(batches, cfg)

	// Save cache
	if cache != nil {
		if err := cache.Save(); err != nil {
			log.Printf("⚠️  Failed to save cache: %v\n", err)
		} else {
			log.Printf("💾 Cache saved (%d entries)\n", cache.Size())
		}
	}

	integrationtest.ReportResults(t, allResults, serviceStatsMap, totalBuildTime, batchStats)
}

// runIndividualPipeline runs tests with individual builds per service (original behavior)
func runIndividualPipeline(t *testing.T, specs []string, sandboxDir string, runtimeOpts *integrationtest.RuntimeOptions) {
	log.Printf("Testing %d services (max %d failures before abort)...\n", len(specs), maxFails)

	// Set up interrupt handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	semaphore := make(chan struct{}, runtimeOpts.MaxConcurrency)
	var wg sync.WaitGroup
	var portCounter atomic.Int32
	var failCount atomic.Int32
	var aborted atomic.Bool
	var completed atomic.Int32

	var resultsMu sync.Mutex
	var allResults []integrationtest.IntegrationResult
	serviceStatsMap := make(map[string]*integrationtest.ServiceStats)

	// Monitor for interrupt signal
	go func() {
		<-sigCh
		log.Printf("\n⚠️  Interrupted! Waiting for in-progress tests to finish...\n")
		aborted.Store(true)
	}()

	for _, specFile := range specs {
		if aborted.Load() {
			log.Printf("⚠️  Aborting remaining services\n")
			break
		}

		semaphore <- struct{}{}
		wg.Add(1)

		go func(spec string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			if aborted.Load() {
				return
			}

			serviceName := api.NormalizeServiceName(spec)
			port := basePort + int(portCounter.Add(1))

			log.Printf("  ⏳ %s\n", serviceName)
			results, stats := runServicePipeline(spec, serviceName, sandboxDir, port, runtimeOpts)

			resultsMu.Lock()
			allResults = append(allResults, results...)
			serviceStatsMap[serviceName] = stats
			resultsMu.Unlock()

			done := int(completed.Add(1))

			for _, r := range results {
				if !r.Ok {
					currentFails := int(failCount.Add(1))
					if currentFails >= maxFails {
						if aborted.CompareAndSwap(false, true) {
							log.Printf("⚠️  Max failures (%d) reached!\n", maxFails)
						}
					}
				}
			}

			status := "✅"
			if stats.Fails > 0 {
				status = "❌"
			}
			log.Printf("  [%d/%d] %s %s (%d/%d ok, %.1fs)\n",
				done, len(specs), status, serviceName, stats.Success, stats.Tested, stats.TTE.Seconds())
		}(specFile)
	}

	wg.Wait()
	integrationtest.ReportResults(t, allResults, serviceStatsMap, 0) // 0 = use TTG+TTC from stats
}

// runServicePipeline runs the full pipeline for a single service: setup → generate → build → start → test → stop
func runServicePipeline(specFile, serviceName, sandboxDir string, port int, runtimeOpts *integrationtest.RuntimeOptions) ([]integrationtest.IntegrationResult, *integrationtest.ServiceStats) {
	emptyStats := &integrationtest.ServiceStats{Name: serviceName}

	// Step 1: Setup
	if !integrationtest.IsServiceSetup(serviceName, sandboxDir) {
		if _, err := integrationtest.SetupService(specFile, sandboxDir, runtimeOpts); err != nil {
			return []integrationtest.IntegrationResult{{
				Spec:        specFile,
				Ok:          false,
				GenerateErr: fmt.Sprintf("Setup failed: %v", err),
			}}, emptyStats
		}
	}

	// Step 2: Generate
	if err := integrationtest.GenerateServiceWithTimeout(sandboxDir, serviceName, runtimeOpts.ServiceGenerateTimeout); err != nil {
		return []integrationtest.IntegrationResult{{
			Spec:        specFile,
			Ok:          false,
			GenerateErr: fmt.Sprintf("Generate failed: %v", err),
		}}, emptyStats
	}

	// Step 3: Build
	serverBin, err := integrationtest.BuildServiceServer(sandboxDir, serviceName)
	if err != nil {
		return []integrationtest.IntegrationResult{{
			Spec:        specFile,
			Ok:          false,
			GenerateErr: fmt.Sprintf("Build failed: %v", err),
		}}, emptyStats
	}

	// Step 4: Start server
	serverCmd, err := integrationtest.StartServiceServer(serverBin, sandboxDir, port)
	if err != nil {
		return []integrationtest.IntegrationResult{{
			Spec:        specFile,
			Ok:          false,
			GenerateErr: fmt.Sprintf("Start failed: %v", err),
		}}, emptyStats
	}
	defer integrationtest.StopServiceServer(serverCmd)

	// Wait for server to be ready
	serverURL := fmt.Sprintf("http://localhost:%d", port)
	if err := integrationtest.WaitForServer(serverURL, integrationtest.ServerReadyTimeout, serverCmd); err != nil {
		return []integrationtest.IntegrationResult{{
			Spec:        specFile,
			Ok:          false,
			GenerateErr: fmt.Sprintf("Server not ready: %v\nstderr: %s", err, serverCmd.GetStderr()),
		}}, emptyStats
	}

	// Step 5: Test
	testStart := time.Now()
	results, totalEndpoints, testedEndpoints := integrationtest.TestService(specFile, serverURL)
	testTime := time.Since(testStart)

	// Count successes and failures
	var success, fails int
	for _, res := range results {
		if res.Ok {
			success++
		} else {
			fails++
		}
	}

	stats := &integrationtest.ServiceStats{
		Name:      serviceName,
		Endpoints: totalEndpoints,
		Tested:    testedEndpoints,
		Success:   success,
		Fails:     fails,
		LOC:       integrationtest.CountServiceLOC(serviceName, sandboxDir),
		TTE:       testTime,
	}

	return results, stats
}
