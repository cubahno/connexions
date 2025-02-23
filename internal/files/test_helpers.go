package files

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _   = runtime.Caller(0)
	testDataPath = filepath.Join(filepath.Dir(b), "..", "", "..", "testdata")
)
