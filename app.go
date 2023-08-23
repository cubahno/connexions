package connexions

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type App struct {
	Router     *Router
	BluePrints []RouteRegister
	Paths      *Paths
	mu         sync.Mutex
}

type Paths struct {
	Base              string
	Resources         string
	Contexts          string
	Docs              string
	Samples           string
	Services          string
	ServicesOpenAPI   string
	ServicesFixedRoot string
	ConfigFile        string
}

func NewPaths(baseDir string) *Paths {
	resDir := filepath.Join(baseDir, "resources")
	svcDir := filepath.Join(resDir, "services")

	return &Paths{
		Base:              baseDir,
		Resources:         resDir,
		Contexts:          filepath.Join(resDir, "contexts"),
		Docs:              filepath.Join(resDir, "docs"),
		Samples:           filepath.Join(resDir, "samples"),
		Services:          svcDir,
		ServicesOpenAPI:   filepath.Join(svcDir, ".openapi"),
		ServicesFixedRoot: filepath.Join(svcDir, ".root"),
		ConfigFile:        filepath.Join(resDir, "config.yml"),
	}
}

func NewApp(baseDir string) *App {
	paths := NewPaths(baseDir)
	res := &App{
		Paths: paths,
	}
	resourcePath := paths.Resources
	log.Printf("Initing Application. ResourcePath is: %v\n", resourcePath)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	err := MustFileStructure(paths)
	if err != nil {
		panic(err)
	}
	_ = CleanupServiceFileStructure(paths.Services)

	config, err := NewConfigFromFile(paths.ConfigFile)
	if err != nil {
		log.Printf("Failed to load config file: %s\n", err.Error())
		config = NewDefaultConfig()
	}

	router := &Router{
		Mux:    r,
		Config: config,
		Paths:  paths,
	}
	res.Router = router

	bluePrints := []RouteRegister{
		LoadServices,
		LoadContexts,

		CreateHomeRoutes,
		CreateServiceRoutes,
		CreateContextRoutes,
		CreateSettingsRoutes,
	}
	res.BluePrints = bluePrints

	for _, bluePrint := range bluePrints {
		err := bluePrint(router)
		if err != nil {
			log.Printf("Failed to load blueprint: %s\n", err.Error())
		}
	}

	return res
}

// MustFileStructure creates the necessary directories and files
func MustFileStructure(paths *Paths) error {
	if _, err := os.Stat(paths.Services); os.IsNotExist(err) {
		log.Println("Creating service directory and configuration for the first time")
		if err := os.Mkdir(paths.Services, os.ModePerm); err != nil {
			return err
		}
	} else {
		return nil
	}

	log.Println("Copying sample content to service directory")
	if err := CopyDirectory(paths.Samples, paths.Services); err != nil {
		return err
	}

	log.Println("Copying sample config file")

	destPath := paths.ConfigFile
	srcPath := fmt.Sprintf("%s.dist", destPath)
	if err := CopyFile(srcPath, destPath); err != nil {
		return err
	}

	log.Println("Done!")
	return nil
}

func (a *App) AddBluePrint(bluePrint RouteRegister) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.BluePrints = append(a.BluePrints, bluePrint)
	err := bluePrint(a.Router)
	if err != nil {
		return err
	}
	return nil
}

func (a *App) Run() {
	config := a.Router.Config
	port := config.App.Port
	homeURL := strings.TrimPrefix(config.App.HomeURL, "/")

	log.Printf("\n\nServer started on port %d. Press Ctrl+C to quit", port)
	log.Printf("Visit http://localhost:%d/%s to view the home page", port, homeURL)

	err := http.ListenAndServe(fmt.Sprintf(":%v", port), a.Router)
	if err != nil {
		panic(err)
	}
}
