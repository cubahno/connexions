package integrationtest

import (
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// RuntimeOptions contains runtime configuration for integration tests.
// All fields contain final computed values with defaults applied.
// Use NewRuntimeOptionsFromEnv() to create with values from .env.dist, .env, and environment.
type RuntimeOptions struct {
	CodegenConfigPath      string        `env:"CODEGEN_CONFIG"`
	ServiceConfigPath      string        `env:"SERVICE_CONFIG"`
	SpecsBaseDir           string        `env:"SPECS_BASE_DIR"`
	MaxConcurrency         int           `env:"MAX_CONCURRENCY" envDefault:"8"`
	BatchSizeMB            int           `env:"BATCH_SIZE_MB" envDefault:"6"`
	BatchConcurrency       int           `env:"BATCH_CONCURRENCY" envDefault:"2"`
	MaxSpecSizeMB          int           `env:"MAX_SPEC_SIZE_MB" envDefault:"10"`
	SimplifyThresholdMB    int           `env:"SIMPLIFY_THRESHOLD_MB" envDefault:"3"`
	ServiceGenerateTimeout time.Duration `env:"SERVICE_GENERATE_TIMEOUT" envDefault:"5m"`
	NoCache                bool          `env:"NO_CACHE" envDefault:"false"`
	ClearCache             bool          `env:"CLEAR_CACHE" envDefault:"false"`
	MaxFails               int           `env:"MAX_FAILS" envDefault:"500"`
	RandomSpecs            int           `env:"RANDOM_SPECS" envDefault:"0"`

	// Computed fields (not from env)
	BatchSizeBytes         int64
	MaxSpecSizeBytes       int64
	SimplifyThresholdBytes int64
}

// NewRuntimeOptionsFromEnv creates RuntimeOptions populated from:
// 1. .env.dist (defaults)
// 2. .env (overrides .env.dist, if exists)
// 3. Environment variables (final overrides - command line takes precedence)
func NewRuntimeOptionsFromEnv() *RuntimeOptions {
	// Load .env.dist first (defaults) - only sets vars not already in env
	_ = godotenv.Load(".env.dist")

	// Load .env to override .env.dist - only sets vars not already in env
	// This means command-line env vars take precedence over .env
	_ = godotenv.Load(".env")

	opts := &RuntimeOptions{}
	if err := env.Parse(opts); err != nil {
		log.Printf("Warning: failed to parse env: %v", err)
	}

	// Compute byte values from MB
	opts.BatchSizeBytes = int64(opts.BatchSizeMB) * 1024 * 1024
	opts.MaxSpecSizeBytes = int64(opts.MaxSpecSizeMB) * 1024 * 1024
	opts.SimplifyThresholdBytes = int64(opts.SimplifyThresholdMB) * 1024 * 1024

	// Use embedded codegen config template if not provided
	if opts.CodegenConfigPath == "" {
		opts.CodegenConfigPath = writeCodegenConfigTemplate()
	}

	return opts
}
