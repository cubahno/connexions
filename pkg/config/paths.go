package config

import (
	"path/filepath"
)

// Paths is a struct that holds all the paths used by the application.
type Paths struct {
	Base      string
	Docs      string
	Resources string
	Data      string
	OpenAPI   string
	Static    string
	Services  string
	UI        string
}

func NewPaths(baseDir string) Paths {
	resDir := filepath.Join(baseDir, "resources")
	dataDir := filepath.Join(resDir, "data")
	svcDir := filepath.Join(dataDir, "services")

	return Paths{
		Base:      baseDir,
		Resources: resDir,
		Data:      dataDir,
		Services:  svcDir,
		OpenAPI:   filepath.Join(dataDir, "openapi"),
		Static:    filepath.Join(dataDir, "static"),

		Docs: filepath.Join(resDir, "docs"),
		UI:   filepath.Join(resDir, "ui"),
	}
}
