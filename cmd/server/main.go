package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cubahno/connexions/internal/api"
	"github.com/cubahno/connexions/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	appDir := os.Getenv("APP_DIR")
	if appDir == "" {
		_, b, _, _ := runtime.Caller(0)
		appDir = filepath.Dir(filepath.Dir(filepath.Dir(b)))
	}
	_ = godotenv.Load(fmt.Sprintf("%s/.env", appDir))

	cfg := config.MustConfig(appDir)
	app := api.NewApp(cfg)
	app.Run()
}
