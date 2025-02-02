package internal

import (
	"path/filepath"
	"runtime"
)

var (
	_, b, _, _   = runtime.Caller(0)
	TestDataPath = filepath.Join(filepath.Dir(b), "..", "testdata")
)
