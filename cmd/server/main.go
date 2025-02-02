package main

import (
	"path/filepath"
	"runtime"

	"github.com/cubahno/connexions/internal"
	"github.com/cubahno/connexions/internal/api"
	"github.com/joho/godotenv"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(b)))
	_ = godotenv.Load()

	cfg := internal.MustConfig(baseDir)
	app := api.NewApp(cfg)
	app.Run()
}
