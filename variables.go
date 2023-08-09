package xs

import (
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

var (
	_, b, _, _   = runtime.Caller(0)
	RootPath     = filepath.Dir(b)
	ResourcePath = fmt.Sprintf("%s/resources", RootPath)
	ServicePath  = fmt.Sprintf("%s/services", ResourcePath)
	UIPath       = fmt.Sprintf("%s/ui", ResourcePath)
	ConfigPath   = fmt.Sprintf("%s/config.yml", ResourcePath)
	LogFlushWait = 200 * time.Millisecond
)
