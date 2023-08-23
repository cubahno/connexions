package connexions

import (
	"fmt"
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _         = runtime.Caller(0)
	RootPath           = filepath.Dir(b)
	ResourcePath       = fmt.Sprintf("%s/resources", RootPath)
	ServicePath        = fmt.Sprintf("%s/services", ResourcePath)
	ContextPath        = fmt.Sprintf("%s/contexts", ResourcePath)
	SamplesPath        = fmt.Sprintf("%s/samples", ResourcePath)
	ServiceOpenAPIPath = fmt.Sprintf("%s/services/.openapi", ResourcePath)
	ServiceRootPath    = fmt.Sprintf("%s/services/.root", ResourcePath)
	UIPath             = fmt.Sprintf("%s/ui", ResourcePath)
	TestSchemaPath     = fmt.Sprintf("%s/test/schemas", ResourcePath)
	ConfigPath         = fmt.Sprintf("%s/config.yml", ResourcePath)
)
