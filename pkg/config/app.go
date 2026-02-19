package config

import (
	"time"
)

// AppConfig is the app configuration.
// Title is the title of the app displayed in the UI.
// Port is the port number to listen on.
// HomeURL is the URL for the UI home page.
// ServiceURL is the URL for the service and resources endpoints in the UI.
// ContextAreaPrefix sets sub-contexts for replacements in path, header or any other supported place.
// DisableUI is a flag whether to disable the UI.
// Paths is the paths to various resource directories.
// Editor is the editor configuration for the UI.
// HistoryDuration is the duration to keep the history of requests in memory.
// Storage configures shared storage for distributed features (e.g., circuit breaker state).
type AppConfig struct {
	Title             string         `yaml:"title"`
	Port              int            `yaml:"port"`
	HomeURL           string         `yaml:"homeURL"`
	ServiceURL        string         `yaml:"serviceURL"`
	ContextAreaPrefix string         `yaml:"contextAreaPrefix"`
	DisableUI         bool           `yaml:"disableUI"`
	Paths             Paths          `yaml:"-"`
	Editor            *EditorConfig  `yaml:"editor"`
	HistoryDuration   time.Duration  `yaml:"historyDuration" env:"ROUTER_HISTORY_DURATION"`
	Storage           *StorageConfig `yaml:"storage"`
}

// NewDefaultAppConfig creates a new default app config in case the config file is missing, not found or any other error.
func NewDefaultAppConfig(baseDir string) *AppConfig {
	return &AppConfig{
		Title:             "Connexions",
		Port:              2200,
		HomeURL:           "/.ui",
		ServiceURL:        "/.services",
		ContextAreaPrefix: "in-",
		Paths:             NewPaths(baseDir),
		Editor: &EditorConfig{
			Theme:    "chrome",
			FontSize: 16,
		},
		HistoryDuration: 5 * time.Minute,
	}
}

type EditorConfig struct {
	Theme    string `yaml:"theme"`
	FontSize int    `yaml:"fontSize"`
}

// StorageType defines the type of storage backend.
type StorageType string

const (
	// StorageTypeMemory is the default in-memory storage (per-instance).
	StorageTypeMemory StorageType = "memory"

	// StorageTypeRedis uses Redis for distributed storage.
	StorageTypeRedis StorageType = "redis"
)

// StorageConfig configures shared storage for distributed features.
type StorageConfig struct {
	Type  StorageType  `yaml:"type"`
	Redis *RedisConfig `yaml:"redis"`
}

// RedisConfig configures Redis connection.
type RedisConfig struct {
	// host:port
	Address  string `yaml:"address"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}
