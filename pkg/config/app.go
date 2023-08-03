package config

import (
	"time"
)

// AppConfig is the app configuration.
// Title is the title of the app displayed in the UI.
// Port is the port number to listen on.
// HomeURL is the URL for the UI home page.
// ServiceURL is the URL for the service and resources endpoints in the UI.
// SettingsURL is the URL for the settings endpoint in the UI.
// ContextURL is the URL for the context endpoint in the UI.
// ContextAreaPrefix sets sub-contexts for replacements in path, header or any other supported place.
//
// for example:
// in-path:
//
//	user_id: "fake:ids.int8"
//
// DisableUI is a flag whether to disable the UI.
// DisableSpec is a flag whether to disable the Swagger UI.
// Paths is the paths to various resource directories.
// CreateFileStructure is a flag whether to create the initial resources file structure:
// contexts, services, etc.
// It will also copy sample files from the samples directory into services.
// Default: true
// HistoryDuration is the duration to keep the history of requests in memory.
//
//	Plugins can access the history.
type AppConfig struct {
	Title             string        `yaml:"title"`
	Port              int           `yaml:"port"`
	HomeURL           string        `yaml:"homeURL"`
	ServiceURL        string        `yaml:"serviceURL"`
	ContextAreaPrefix string        `yaml:"contextAreaPrefix"`
	DisableUI         bool          `yaml:"disableUI"`
	Paths             Paths         `yaml:"-"`
	Editor            *EditorConfig `yaml:"editor"`
	HistoryDuration   time.Duration `yaml:"historyDuration" env:"ROUTER_HISTORY_DURATION"`
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
