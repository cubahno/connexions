package config

import "github.com/cubahno/connexions/internal/types"

// AppConfig is the app configuration.
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
type AppConfig struct {
	Port                int           `json:"port" yaml:"port" koanf:"port"`
	HomeURL             string        `json:"homeUrl" yaml:"homeURL" koanf:"homeUrl"`
	ServiceURL          string        `json:"serviceUrl" yaml:"serviceURL" koanf:"serviceUrl"`
	SettingsURL         string        `json:"settingsUrl" yaml:"settingsURL" koanf:"settingsUrl"`
	ContextURL          string        `json:"contextUrl" yaml:"contextUrl" koanf:"contextUrl"`
	ContextAreaPrefix   string        `json:"contextAreaPrefix" yaml:"contextAreaPrefix" koanf:"contextAreaPrefix"`
	DisableUI           bool          `json:"disableUI" yaml:"disableUI" koanf:"disableUI"`
	DisableSwaggerUI    bool          `json:"disableSwaggerUI" yaml:"disableSwaggerUI" koanf:"disableSwaggerUI"`
	Paths               *Paths        `json:"-" koanf:"-"`
	CreateFileStructure bool          `koanf:"createFileStructure" json:"createFileStructure" yaml:"createFileStructure"`
	Editor              *EditorConfig `koanf:"editor" json:"editor" yaml:"editor"`
}

// IsValidPrefix returns true if the prefix is not a reserved URL.
func (a *AppConfig) IsValidPrefix(prefix string) bool {
	return !types.SliceContains([]string{
		a.HomeURL,
		a.ServiceURL,
		a.SettingsURL,
		a.ContextURL,
	}, prefix)
}

// NewDefaultAppConfig creates a new default app config in case the config file is missing, not found or any other error.
func NewDefaultAppConfig(baseDir string) *AppConfig {
	return &AppConfig{
		Port:              2200,
		HomeURL:           "/.ui",
		ServiceURL:        "/.services",
		SettingsURL:       "/.settings",
		ContextURL:        "/.contexts",
		ContextAreaPrefix: "in-",
		Paths:             NewPaths(baseDir),
		Editor: &EditorConfig{
			Theme:    "chrome",
			FontSize: 16,
		},
	}
}
