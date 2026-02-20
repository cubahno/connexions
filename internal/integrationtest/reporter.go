package integrationtest

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// ServiceStats holds statistics for a single service
type ServiceStats struct {
	Name      string
	Endpoints int
	Tested    int
	Success   int
	Fails     int
	LOC       int
	TTE       time.Duration // Time To Execute (test execution)
}

// ReportResults prints a summary of test results
func ReportResults(t *testing.T, results []IntegrationResult, serviceStatsMap map[string]*ServiceStats, totalBuildTime time.Duration, batchStats ...*BatchStats) {
	success := 0
	fails := 0
	generateFails := 0

	var failedResults []IntegrationResult

	for _, res := range results {
		if res.Ok {
			success++
		} else {
			fails++
			failedResults = append(failedResults, res)

			if res.GenerateErr != "" {
				generateFails++
			}
		}
	}

	fmt.Fprintf(os.Stderr, "\n=== Integration Test Results ===\n")
	fmt.Fprintf(os.Stderr, "Total operations tested: %d\n", len(results))
	fmt.Fprintf(os.Stderr, "✅ Success: %d\n", success)
	fmt.Fprintf(os.Stderr, "❌ Fails: %d\n", fails)
	if fails > 0 {
		fmt.Fprintf(os.Stderr, "  - Generate failures: %d\n", generateFails)
	}
	fmt.Fprintf(os.Stderr, "========================================\n\n")

	// Print service summary in alphabetical order
	if len(serviceStatsMap) > 0 {
		fmt.Fprintf(os.Stderr, "\n=== Services Summary ===\n")

		// Convert map to slice and sort alphabetically
		var services []*ServiceStats
		for _, stats := range serviceStatsMap {
			services = append(services, stats)
		}
		sort.Slice(services, func(i, j int) bool {
			return services[i].Name < services[j].Name
		})

		totalEndpoints := 0
		totalTested := 0
		totalSuccess := 0
		totalFails := 0
		totalTTE := time.Duration(0)

		totalLOC := 0
		for _, stats := range services {
			locStr := formatLOC(stats.LOC)
			fmt.Fprintf(os.Stderr, "  %-52s  ✅ %3d  ❌ %3d  (%3d endpoints, %s LOC)  TTE: %7s\n",
				stats.Name, stats.Success, stats.Fails, stats.Endpoints, locStr,
				stats.TTE.Round(time.Millisecond))
			totalEndpoints += stats.Endpoints
			totalTested += stats.Tested
			totalSuccess += stats.Success
			totalFails += stats.Fails
			totalTTE += stats.TTE
			totalLOC += stats.LOC
		}

		totalLOCStr := formatLOC(totalLOC)

		// Use passed totalBuildTime if available (batch mode)
		buildTime := totalBuildTime
		totalTime := buildTime + totalTTE
		var avgPerSpec time.Duration
		if len(services) > 0 {
			avgPerSpec = totalTime / time.Duration(len(services))
		}

		fmt.Fprintf(os.Stderr, "\n  Total: %d services, %d endpoints (tested: %d), %s LOC\n", len(services), totalEndpoints, totalTested, totalLOCStr)
		fmt.Fprintf(os.Stderr, "         ✅ Success: %d  ❌ Fails: %d\n", totalSuccess, totalFails)
		fmt.Fprintf(os.Stderr, "         ⏱️  Build: %s  Test: %s  Total: %s  Avg/spec: %s\n",
			buildTime.Round(time.Millisecond), totalTTE.Round(time.Millisecond),
			totalTime.Round(time.Millisecond), avgPerSpec.Round(time.Millisecond))

		// Print batch stats if available
		if len(batchStats) > 0 && batchStats[0] != nil {
			bs := batchStats[0]
			fmt.Fprintf(os.Stderr, "         📦 Batches: %d  Avg build: %s  Avg test: %s\n",
				bs.TotalBatches, bs.AvgBuildTime().Round(time.Millisecond), bs.AvgTestTime().Round(time.Millisecond))
		}
		fmt.Fprintf(os.Stderr, "========================================\n\n")
	}

	// If there are failures, print them
	if len(failedResults) > 0 {
		t.Errorf("Failed %d out of %d operation tests", fails, len(results))

		// Separate actual failures from batch-level failures
		var actualFailures []IntegrationResult
		var batchFailures []IntegrationResult
		for _, res := range failedResults {
			errMsg := res.GenerateErr
			// Batch-level failures (server didn't start, interrupted, connection refused after kill, etc.)
			if strings.HasPrefix(errMsg, "skipped:") ||
				strings.Contains(errMsg, "server not ready") ||
				strings.Contains(errMsg, "interrupted") ||
				strings.Contains(errMsg, "connection refused") {
				batchFailures = append(batchFailures, res)
				continue
			}
			actualFailures = append(actualFailures, res)
		}

		// Print detailed failure information
		fmt.Fprintf(os.Stderr, "\n=== Failure Details ===\n")
		for i, res := range actualFailures {
			fmt.Fprintf(os.Stderr, "\n%d. %s %s\n", i+1, res.Method, res.Path)
			if res.OperationID != "" {
				fmt.Fprintf(os.Stderr, "   Operation ID: %s\n", res.OperationID)
			}
			fmt.Fprintf(os.Stderr, "   Spec: %s\n", res.Spec)
			if res.BatchID > 0 {
				fmt.Fprintf(os.Stderr, "   Batch: %d\n", res.BatchID)
			}
			if res.GenerateErr != "" {
				fmt.Fprintf(os.Stderr, "   Error: %s\n", res.GenerateErr)
			}
			if res.StatusCode > 0 {
				fmt.Fprintf(os.Stderr, "   Status Code: %d\n", res.StatusCode)
			}
			if res.Validated {
				fmt.Fprintf(os.Stderr, "   Validated: true (method: %s)\n", res.ValidationMethod)
			}
		}

		// Print batch failures (grouped by error to avoid repetition)
		if len(batchFailures) > 0 {
			fmt.Fprintf(os.Stderr, "\n=== Batch Failures (server failed to start) ===\n")
			// Group specs by error message (same batch = same error)
			errorSpecs := make(map[string][]string)
			for _, res := range batchFailures {
				errorSpecs[res.GenerateErr] = append(errorSpecs[res.GenerateErr], res.Spec)
			}
			for errMsg, specs := range errorSpecs {
				if len(specs) == 1 {
					fmt.Fprintf(os.Stderr, "   %s: %s\n", specs[0], errMsg)
				} else {
					fmt.Fprintf(os.Stderr, "   [%d specs in batch]: %s\n", len(specs), errMsg)
					for _, spec := range specs {
						fmt.Fprintf(os.Stderr, "      - %s\n", spec)
					}
				}
			}
		}
		fmt.Fprintf(os.Stderr, "========================================\n\n")

		// Print sorted list of spec files with failures
		fmt.Fprintf(os.Stderr, "=== Specs with Failures ===\n")

		// Collect unique spec files with failures (both actual and batch)
		failedSpecs := make(map[string]bool)
		for _, res := range actualFailures {
			failedSpecs[res.Spec] = true
		}
		for _, res := range batchFailures {
			failedSpecs[res.Spec] = true
		}

		// Sort spec files
		var specList []string
		for spec := range failedSpecs {
			specList = append(specList, spec)
		}
		sort.Strings(specList)

		for _, spec := range specList {
			fmt.Fprintf(os.Stderr, "  %s\n", spec)
		}
		fmt.Fprintf(os.Stderr, "========================================\n\n")
	}
}
