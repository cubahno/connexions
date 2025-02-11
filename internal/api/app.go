package api

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cubahno/connexions/internal/config"
	"github.com/cubahno/connexions/internal/types"
)

// App is the main application struct
type App struct {
	Router *Router
	Paths  *config.Paths

	bluePrints []RouteRegister
	mu         sync.Mutex
}

// NewApp creates a new App instance from Config and registers predefined blueprints.
func NewApp(config *config.Config) *App {
	paths := config.App.Paths
	res := &App{
		Paths: paths,
	}
	resourcePath := paths.Resources
	log.Printf("Initing Application. ResourcePath is: %v\n", resourcePath)

	// Seed the random number generator
	rand.New(rand.NewSource(time.Now().UnixNano()))

	if config.App.CreateFileStructure {
		err := MustFileStructure(paths)
		if err != nil {
			panic(err)
		}
		_ = types.CleanupServiceFileStructure(paths.Services)
	}

	router := NewRouter(config)
	res.Router = router

	bluePrints := []RouteRegister{
		loadMiddleware,
		loadContexts,
		loadServices,

		createHealthRoutes,
		createHomeRoutes,
		createServiceRoutes,
		createContextRoutes,
		createSettingsRoutes,
	}
	res.bluePrints = bluePrints

	for _, bluePrint := range bluePrints {
		err := bluePrint(router)
		if err != nil {
			log.Printf("Failed to load blueprint: %s\n", err.Error())
		}
	}

	return res
}

// MustFileStructure creates the necessary directories and files
func MustFileStructure(paths *config.Paths) error {
	dirs := []string{paths.Resources, paths.Samples, paths.Data, paths.Services, paths.Contexts, paths.Middleware}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.Mkdir(dir, os.ModePerm); err != nil {
				return err
			}
		}
	}

	log.Println("Done!")
	return nil
}

// AddBluePrint adds a new blueprint to the application.
func (a *App) AddBluePrint(bluePrint RouteRegister) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.bluePrints = append(a.bluePrints, bluePrint)
	err := bluePrint(a.Router)
	if err != nil {
		return err
	}
	return nil
}

// Run starts the application and the server.
// Blocks until the server is stopped.
func (a *App) Run() {
	defer func() {
		a.Router.history.Cancel()
	}()
	cfg := a.Router.Config
	port := cfg.App.Port
	homeURL := strings.TrimPrefix(cfg.App.HomeURL, "/")

	log.Printf("\n\nServer started on port %d. Press Ctrl+C to quit", port)
	log.Printf("Visit http://localhost:%d/%s to view the home page", port, homeURL)

	err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), a.Router)
	if err != nil {
		panic(err)
	}
}
