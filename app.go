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
	mu         sync.Mutex
}

func NewApp() *App {
	res := &App{}
	log.Printf("Initing Application. ResourcePath is: %v\n", ResourcePath)

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	err := MustFileStructure()
	if err != nil {
		panic(err)
	}
	_ = CleanupFileStructure()

	config, err := NewConfigFromFile(fmt.Sprintf("%s/config.yml", ResourcePath))
	if err != nil {
		log.Printf("Failed to load config file: %s\n", err.Error())
		config = NewDefaultConfig()
	}

	router := &Router{
		Mux:    r,
		Config: config,
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
func MustFileStructure() error {
	if _, err := os.Stat(ServicePath); os.IsNotExist(err) {
		log.Print("Creating service directory and configuration for the first time")
		if err := os.Mkdir(ServicePath, os.ModePerm); err != nil {
			return err
		}
	} else {
		return nil
	}

	log.Print("Copying sample content to service directory")
	if err := CopyDirectory(SamplesPath, ServicePath); err != nil {
		return err
	}

	log.Print("Copying sample config file")
	srcPath := filepath.Join(ResourcePath, "config.yml.dist")
	destPath := filepath.Join(ResourcePath, "config.yml")
	if err := CopyFile(srcPath, destPath); err != nil {
		return err
	}

	log.Print("Done!")
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
