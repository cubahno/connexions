package main

import (
	"github.com/cubahno/connexions/api"
	"github.com/cubahno/connexions/config"
	"github.com/joho/godotenv"
	"path/filepath"
	"runtime"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(b)))
	_ = godotenv.Load()

	cfg := config.MustConfig(baseDir)
	app := api.NewApp(cfg)
	app.Run()
}
