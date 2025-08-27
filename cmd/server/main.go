package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cubahno/connexions/internal/api"
	"github.com/cubahno/connexions/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

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
