package config

import "path/filepath"

// Paths is a struct that holds all the paths used by the application.
type Paths struct {
	Base              string
	Resources         string
	Data              string
	Middleware        string
	Contexts          string
	Docs              string
	Samples           string
	Services          string
	ServicesOpenAPI   string
	ServicesFixedRoot string
	UI                string
	ConfigFile        string
}

func NewPaths(baseDir string) *Paths {
	resDir := filepath.Join(baseDir, "resources")
	dataDir := filepath.Join(resDir, "data")
	svcDir := filepath.Join(dataDir, "services")
	mwDir := filepath.Join(dataDir, "middleware")

	return &Paths{
		Base:      baseDir,
		Resources: resDir,

		Data:              dataDir,
		Middleware:        mwDir,
		Contexts:          filepath.Join(dataDir, "contexts"),
		ConfigFile:        filepath.Join(dataDir, "config.yml"),
		Services:          svcDir,
		ServicesOpenAPI:   filepath.Join(svcDir, RootOpenAPIName),
		ServicesFixedRoot: filepath.Join(svcDir, RootServiceName),

		Docs:    filepath.Join(resDir, "docs"),
		Samples: filepath.Join(resDir, "samples"),
		UI:      filepath.Join(resDir, "ui"),
	}
}
