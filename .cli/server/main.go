package main

import (
	"github.com/cubahno/connexions"
	"path/filepath"
	"runtime"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(b)))

	app := connexions.NewApp(baseDir)
	app.Run()
}
