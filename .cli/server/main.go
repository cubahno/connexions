package main

import (
	"github.com/cubahno/connexions"
	"path/filepath"
	"runtime"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(b)))

	config := connexions.MustConfig(baseDir)
	app := connexions.NewApp(config)
	app.Run()
}
