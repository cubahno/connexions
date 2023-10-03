package main

import (
	"github.com/cubahno/connexions"
	"github.com/cubahno/connexions/api"
	"github.com/joho/godotenv"
	"path/filepath"
	"runtime"
)

func main() {
	_, b, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Dir(b)))
	_ = godotenv.Load()

	config := connexions.MustConfig(baseDir)
	app := api.NewApp(config)
	app.Run()
}
